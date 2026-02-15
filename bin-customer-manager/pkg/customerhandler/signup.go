package customerhandler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-customer-manager/internal/config"
	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/cachehandler"
	"monorepo/bin-customer-manager/pkg/metricshandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const (
	emailVerifyTokenTTL = time.Hour
	emailVerifyTokenLen = 32 // 32 bytes = 64 hex chars
	tempTokenLen        = 16 // 16 bytes = 32 hex chars
	maxSignupAttempts   = 5
)

func cryptoRandInt(min, max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min)))
	if err != nil {
		return 0, fmt.Errorf("could not generate random number: %w", err)
	}
	return int(n.Int64()) + min, nil
}

// Signup creates an unverified customer and sends a verification email.
func (h *customerHandler) Signup(
	ctx context.Context,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod customer.WebhookMethod,
	webhookURI string,
) (*customer.SignupResult, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "Signup",
		"email": email,
	})
	log.Debug("Processing customer signup.")

	// validate email
	if !h.validateCreate(ctx, email) {
		log.Errorf("Email validation failed. email: %s", email)
		metricshandler.SignupTotal.WithLabelValues("validation_failed").Inc()
		return nil, fmt.Errorf("the email is not available")
	}

	id := h.utilHandler.UUIDCreate()

	// create customer with email_verified = false
	u := &customer.Customer{
		ID: id,

		Name:   name,
		Detail: detail,

		Email:       email,
		PhoneNumber: phoneNumber,
		Address:     address,

		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,

		EmailVerified: false,
	}

	if err := h.db.CustomerCreate(ctx, u); err != nil {
		log.Errorf("Could not create customer. err: %v", err)
		metricshandler.SignupTotal.WithLabelValues("error").Inc()
		return nil, err
	}

	res, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created customer. err: %v", err)
		metricshandler.SignupTotal.WithLabelValues("error").Inc()
		return nil, err
	}
	log.WithField("customer", res).Debugf("Created unverified customer. customer_id: %s", res.ID)

	// generate verification token
	tokenBytes := make([]byte, emailVerifyTokenLen)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Errorf("Could not generate random token. err: %v", err)
		metricshandler.SignupTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("could not generate verification token")
	}
	token := hex.EncodeToString(tokenBytes)

	// store token in Redis
	if err := h.cache.EmailVerifyTokenSet(ctx, token, id, emailVerifyTokenTTL); err != nil {
		log.Errorf("Could not store verification token. err: %v", err)
		metricshandler.SignupTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("could not store verification token")
	}

	// generate OTP code (100000..999999 inclusive)
	otpNum, err := cryptoRandInt(100000, 1000000)
	if err != nil {
		log.Errorf("Could not generate OTP code. err: %v", err)
		metricshandler.SignupTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("could not generate verification code")
	}
	otpCode := fmt.Sprintf("%06d", otpNum)

	// generate temp_token
	tempTokenBytes := make([]byte, tempTokenLen)
	if _, err := rand.Read(tempTokenBytes); err != nil {
		log.Errorf("Could not generate temp token. err: %v", err)
		metricshandler.SignupTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("could not generate temp token")
	}
	tempToken := hex.EncodeToString(tempTokenBytes)

	// store signup session in Redis
	session := &cachehandler.SignupSession{
		CustomerID:  id,
		OTPCode:     otpCode,
		VerifyToken: token,
	}
	if err := h.cache.SignupSessionSet(ctx, tempToken, session, emailVerifyTokenTTL); err != nil {
		log.Errorf("Could not store signup session. err: %v", err)
		metricshandler.SignupTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("could not store signup session")
	}

	// send verification email
	if err := h.sendVerificationEmail(ctx, email, token, otpCode); err != nil {
		log.Errorf("Could not send verification email. err: %v", err)
		// still return the customer even if email sending fails
	}

	// do NOT publish customer_created event — wait for email verification

	metricshandler.SignupTotal.WithLabelValues("success").Inc()

	return &customer.SignupResult{Customer: res, TempToken: tempToken}, nil
}

// EmailVerify validates a verification token and activates the customer.
func (h *customerHandler) EmailVerify(ctx context.Context, token string) (*customer.EmailVerifyResult, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "EmailVerify",
	})
	log.Debug("Processing email verification.")

	// look up token in Redis
	customerID, err := h.cache.EmailVerifyTokenGet(ctx, token)
	if err != nil {
		log.Errorf("Could not get verification token. err: %v", err)
		metricshandler.EmailVerificationTotal.WithLabelValues("invalid_token").Inc()
		return nil, fmt.Errorf("verification token expired or invalid")
	}
	log.Debugf("Found verification token. customer_id: %s", customerID)

	// get customer
	c, err := h.db.CustomerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get customer. err: %v", err)
		metricshandler.EmailVerificationTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("customer not found")
	}
	log.WithField("customer", c).Debugf("Retrieved customer info. customer_id: %s", c.ID)

	if c.EmailVerified {
		log.Infof("Customer already verified. customer_id: %s", c.ID)
		metricshandler.EmailVerificationTotal.WithLabelValues("already_verified").Inc()
		return &customer.EmailVerifyResult{Customer: c}, nil
	}

	// mark as verified
	fields := map[customer.Field]any{
		customer.FieldEmailVerified: true,
	}
	if err := h.db.CustomerUpdate(ctx, customerID, fields); err != nil {
		log.Errorf("Could not update customer. err: %v", err)
		metricshandler.EmailVerificationTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("could not verify customer")
	}

	// delete token (single-use)
	if err := h.cache.EmailVerifyTokenDelete(ctx, token); err != nil {
		log.Errorf("Could not delete verification token. err: %v", err)
		// not fatal, continue
	}

	// get updated customer
	res, err := h.db.CustomerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get updated customer. err: %v", err)
		metricshandler.EmailVerificationTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("could not get verified customer")
	}
	log.WithField("customer", res).Debugf("Customer verified. customer_id: %s", res.ID)

	// Create AccessKey (auto-provisioning)
	ak, err := h.accesskeyHandler.Create(ctx, customerID, "default", "Auto-provisioned API key", 0)
	if err != nil {
		log.Errorf("Could not create access key during email verify. err: %v", err)
		// Non-fatal — customer is verified but key creation failed
	}

	// publish customer_created event with headless=false
	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerCreated, &customer.CustomerCreatedEvent{
		Customer: res,
		Headless: false,
	})

	metricshandler.EmailVerificationTotal.WithLabelValues("success").Inc()

	return &customer.EmailVerifyResult{
		Customer:  res,
		Accesskey: ak,
	}, nil
}

// CompleteSignup validates an OTP code and completes the headless signup flow.
func (h *customerHandler) CompleteSignup(ctx context.Context, tempToken string, code string) (*customer.CompleteSignupResult, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "CompleteSignup",
	})
	log.Debug("Processing headless signup completion.")

	// Rate limit check
	count, err := h.cache.SignupAttemptIncrement(ctx, tempToken, emailVerifyTokenTTL)
	if err != nil {
		log.Errorf("Could not increment attempt counter. err: %v", err)
		metricshandler.CompleteSignupTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("internal error")
	}
	if count > maxSignupAttempts {
		log.Infof("Too many attempts for temp_token.")
		metricshandler.CompleteSignupTotal.WithLabelValues("rate_limited").Inc()
		return nil, fmt.Errorf("too many attempts")
	}

	// Get signup session from Redis
	session, err := h.cache.SignupSessionGet(ctx, tempToken)
	if err != nil {
		log.Errorf("Could not get signup session. err: %v", err)
		metricshandler.CompleteSignupTotal.WithLabelValues("invalid_token").Inc()
		return nil, fmt.Errorf("invalid or expired temp_token")
	}
	log.Debugf("Found signup session. customer_id: %s", session.CustomerID)

	// Validate OTP
	if session.OTPCode != code {
		log.Infof("Invalid OTP code.")
		metricshandler.CompleteSignupTotal.WithLabelValues("invalid_code").Inc()
		return nil, fmt.Errorf("invalid verification code")
	}

	// Mark customer as verified
	fields := map[customer.Field]any{
		customer.FieldEmailVerified: true,
	}
	if err := h.db.CustomerUpdate(ctx, session.CustomerID, fields); err != nil {
		log.Errorf("Could not update customer. err: %v", err)
		metricshandler.CompleteSignupTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("could not verify customer")
	}

	// Create AccessKey BEFORE cleaning up Redis keys, so if this fails
	// the user can retry with the same temp_token and OTP code.
	ak, err := h.accesskeyHandler.Create(ctx, session.CustomerID, "default", "Auto-provisioned API key", 0)
	if err != nil {
		log.Errorf("Could not create access key. err: %v", err)
		metricshandler.CompleteSignupTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("could not create access key")
	}

	// Delete all Redis keys (session + attempts + email verify token)
	_ = h.cache.SignupSessionDelete(ctx, tempToken)
	_ = h.cache.SignupAttemptDelete(ctx, tempToken)
	_ = h.cache.EmailVerifyTokenDelete(ctx, session.VerifyToken)
	log.WithField("accesskey", ak).Debugf("Created access key. accesskey_id: %s", ak.ID)

	// Get verified customer for event publishing
	cu, err := h.db.CustomerGet(ctx, session.CustomerID)
	if err != nil {
		log.Errorf("Could not get verified customer. err: %v", err)
	}

	// Publish customer_created event with headless=true
	if cu != nil {
		h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerCreated, &customer.CustomerCreatedEvent{
			Customer: cu,
			Headless: true,
		})
	}

	metricshandler.CompleteSignupTotal.WithLabelValues("success").Inc()

	return &customer.CompleteSignupResult{
		CustomerID: session.CustomerID.String(),
		Accesskey:  ak,
	}, nil
}

// sendVerificationEmail sends a verification email to the customer.
func (h *customerHandler) sendVerificationEmail(ctx context.Context, email string, token string, otpCode string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "sendVerificationEmail",
		"email": email,
	})

	cfg := config.Get()
	verifyLink := cfg.EmailVerifyBaseURL + "/auth/email-verify?token=" + token

	subject := fmt.Sprintf("VoIPBin - Verify Your Email (Code: %s)", otpCode)
	content := fmt.Sprintf(
		"Welcome to VoIPBin!\n\n"+
			"Your verification code is: %s\n\n"+
			"API Users: POST this code with your temp_token to /v1/auth/complete-signup\n\n"+
			"Or click the link below to verify via browser (expires in 1 hour):\n\n"+
			"%s\n\n"+
			"If you did not create this account, you can safely ignore this email.",
		otpCode,
		verifyLink,
	)

	destinations := []commonaddress.Address{
		{
			Type:   commonaddress.TypeEmail,
			Target: email,
		},
	}

	start := time.Now()
	rpcStatus := "error"
	defer func() {
		metricshandler.RPCCallDuration.WithLabelValues("email-manager", "email_send").Observe(float64(time.Since(start).Milliseconds()))
		metricshandler.RPCCallTotal.WithLabelValues("email-manager", "email_send", rpcStatus).Inc()
	}()

	if _, err := h.reqHandler.EmailV1EmailSend(
		ctx,
		uuid.Nil,
		uuid.Nil,
		destinations,
		subject,
		content,
		nil,
	); err != nil {
		log.Errorf("Could not send verification email. err: %v", err)
		return err
	}

	rpcStatus = "success"

	log.Debugf("Sent verification email. email: %s", email)
	return nil
}
