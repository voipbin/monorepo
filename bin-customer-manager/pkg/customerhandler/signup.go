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
	emailVerifyTokenTTL = time.Hour
	emailVerifyTokenLen = 32 // 32 bytes = 64 hex chars
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
) (*customer.Customer, error) {
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

	// send verification email
	if err := h.sendVerificationEmail(ctx, email, token); err != nil {
		log.Errorf("Could not send verification email. err: %v", err)
		// still return the customer even if email sending fails
	}

	// do NOT publish customer_created event — wait for email verification

	metricshandler.SignupTotal.WithLabelValues("success").Inc()

	return res, nil
}

// EmailVerify validates a verification token and activates the customer.
func (h *customerHandler) EmailVerify(ctx context.Context, token string) (*customer.Customer, error) {
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
		return c, nil
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

	// publish customer_created event — triggers default agent creation + welcome email
	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerCreated, res)
	metricshandler.EventPublishTotal.WithLabelValues(customer.EventTypeCustomerCreated).Inc()

	metricshandler.EmailVerificationTotal.WithLabelValues("success").Inc()

	return res, nil
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
			"Click the link below to verify your email address. This link expires in 1 hour.\n\n"+
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
