package customerhandler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-customer-manager/internal/config"
	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/metricshandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const (
	emailVerifyTokenTTL    = time.Hour
	emailVerifyTokenLen    = 32 // 32 bytes = 64 hex chars
	defaultAccesskeyExpire = 365 * 24 * time.Hour // 1 year
)

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
	clientIP string,
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

		EmailVerified:              false,
		Status:                     customer.StatusInitial,
		IdentityVerificationStatus: customer.IdentityVerificationStatusNone,

		TermsAgreedVersion: time.Now().UTC().Format(time.RFC3339),
		TermsAgreedIP:      clientIP,
	}
	log.Debugf("Recording terms agreement. client_ip: %s", clientIP)

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

	// Create AccessKey at signup time so headless clients can authenticate immediately
	ak, err := h.accesskeyHandler.Create(ctx, id, "default", "Auto-provisioned API key", defaultAccesskeyExpire)
	if err != nil {
		log.Errorf("Could not create access key during signup. err: %v", err)
		metricshandler.SignupTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("could not create access key")
	}
	log.WithField("accesskey", ak).Debugf("Created access key. accesskey_id: %s", ak.ID)

	// Publish customer_created event with headless=true at signup time.
	// This triggers downstream resource creation (billing, agent, storage).
	// Must only be published here — billing-manager and storage-manager have no idempotency guards.
	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerCreated, &customer.CustomerCreatedEvent{
		Customer: res,
		Headless: true,
	})

	// Best-effort verification email: generate token, store in Redis, and send email.
	// Failures here are non-fatal because the customer and access key are already committed
	// and cannot be rolled back. The client needs the SignupResult to authenticate.
	// If the verification email fails, the customer can re-request signup to trigger a new email.
	if err := h.sendSignupVerification(ctx, id, email); err != nil {
		log.Errorf("Could not complete verification email flow. err: %v", err)
	}

	metricshandler.SignupTotal.WithLabelValues("success").Inc()

	return &customer.SignupResult{Customer: res, Accesskey: ak}, nil
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

	// Acquire verification lock to prevent concurrent verification race
	locked, err := h.cache.VerifyLockAcquire(ctx, customerID, 30*time.Second)
	if err != nil {
		log.Errorf("Could not acquire verify lock. err: %v", err)
		metricshandler.EmailVerificationTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("internal error")
	}
	if !locked {
		log.Infof("Verification already in progress. customer_id: %s", customerID)
		metricshandler.EmailVerificationTotal.WithLabelValues("already_verified").Inc()
		return nil, fmt.Errorf("verification already in progress")
	}
	defer func() { _ = h.cache.VerifyLockRelease(ctx, customerID) }()

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
		// Clean up token (single-use, prevent reuse)
		_ = h.cache.EmailVerifyTokenDelete(ctx, token)
		return &customer.EmailVerifyResult{Customer: c}, nil
	}

	// mark as verified and activate
	fields := map[customer.Field]any{
		customer.FieldEmailVerified: true,
		customer.FieldStatus:        string(customer.StatusActive),
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
	updated, err := h.db.CustomerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get updated customer. err: %v", err)
		metricshandler.EmailVerificationTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("could not get verified customer")
	}
	log.WithField("customer", updated).Debugf("Customer verified. customer_id: %s", updated.ID)

	// Send password reset email so browser users can set their admin password.
	// Non-fatal — customer is verified even if this fails.
	if err := h.reqHandler.AgentV1PasswordForgot(ctx, 30000, updated.Email); err != nil {
		log.Errorf("Could not send password reset email. err: %v", err)
	}

	metricshandler.EmailVerificationTotal.WithLabelValues("success").Inc()

	return &customer.EmailVerifyResult{
		Customer: updated,
	}, nil
}

// sendSignupVerification generates a verification token, stores it in Redis, and sends the verification email.
func (h *customerHandler) sendSignupVerification(ctx context.Context, customerID uuid.UUID, email string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "sendSignupVerification",
		"email": email,
	})

	tokenBytes := make([]byte, emailVerifyTokenLen)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Errorf("Could not generate random token. err: %v", err)
		return fmt.Errorf("could not generate verification token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	if err := h.cache.EmailVerifyTokenSet(ctx, token, customerID, emailVerifyTokenTTL); err != nil {
		log.Errorf("Could not store verification token. err: %v", err)
		return fmt.Errorf("could not store verification token: %w", err)
	}

	if err := h.sendVerificationEmail(ctx, email, token); err != nil {
		log.Errorf("Could not send verification email. err: %v", err)
		return err
	}

	return nil
}

// sendVerificationEmail sends a verification email to the customer.
func (h *customerHandler) sendVerificationEmail(ctx context.Context, email string, token string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "sendVerificationEmail",
		"email": email,
	})

	cfg := config.Get()
	verifyLink := cfg.EmailVerifyBaseURL + "/auth/email-verify?token=" + token

	subject := "VoIPBin - Verify Your Email"
	content := fmt.Sprintf(
		"Welcome to VoIPBin!\n\n"+
			"Click the link below to verify your email address (expires in 1 hour):\n\n"+
			"%s\n\n"+
			"If you did not create this account, you can safely ignore this email.",
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
		customer.IDSystem,
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
