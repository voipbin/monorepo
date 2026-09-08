package service

import (
	"fmt"
	"regexp"

	cscustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var validVerifyToken = regexp.MustCompile(`^[0-9a-f]{64}$`)

// RequestBodySignupPOST is request body for POST /auth/signup
type RequestBodySignupPOST struct {
	Name          string `json:"name"`
	Detail        string `json:"detail"`
	Email         string `json:"email" binding:"required"`
	PhoneNumber   string `json:"phone_number"`
	Address       string `json:"address"`
	WebhookMethod string `json:"webhook_method"`
	WebhookURI    string `json:"webhook_uri"`
	AcceptedTOS   *bool  `json:"accepted_tos" binding:"required"`
}

// PostCustomerSignup handles POST /auth/signup request.
// It always returns 200 to prevent email enumeration.
func PostCustomerSignup(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCustomerSignup",
		"request_address": c.ClientIP(),
	})

	var req RequestBodySignupPOST
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	if !*req.AcceptedTOS {
		log.Infof("Terms of service not accepted.")
		c.AbortWithStatus(400)
		return
	}

	log = log.WithFields(logrus.Fields{
		"email": req.Email,
	})
	log.Debugf("Processing customer signup. email: %s", req.Email)

	sh := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	res, err := sh.CustomerSignup(
		c.Request.Context(),
		req.Name,
		req.Detail,
		req.Email,
		req.PhoneNumber,
		req.Address,
		cscustomer.WebhookMethod(req.WebhookMethod),
		req.WebhookURI,
		c.ClientIP(),
	)
	if err != nil {
		log.Debugf("Customer signup failed. err: %v", err)
		// Return 200 with empty body to prevent email enumeration
		c.JSON(200, gin.H{})
		return
	}

	c.JSON(200, res)
}

// RequestBodyEmailVerifyPOST is request body for POST /auth/email-verify
type RequestBodyEmailVerifyPOST struct {
	Token string `json:"token" binding:"required"`
}

// PostCustomerEmailVerify handles POST /auth/email-verify request.
func PostCustomerEmailVerify(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCustomerEmailVerify",
		"request_address": c.ClientIP(),
	})
	log.Debug("Processing email verification.")

	var req RequestBodyEmailVerifyPOST
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	sh := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	res, err := sh.CustomerEmailVerify(c.Request.Context(), req.Token)
	if err != nil {
		log.Debugf("Email verification failed. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// RequestBodyEmailVerifyResendPOST is request body for POST /auth/email-verify-resend
type RequestBodyEmailVerifyResendPOST struct {
	Email string `json:"email" binding:"required"`
}

// PostCustomerEmailVerifyResend handles POST /auth/email-verify-resend request.
// It always returns 200, including on a malformed body, so that no response
// distinguishes a known address from an unknown one.
func PostCustomerEmailVerifyResend(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostCustomerEmailVerifyResend",
		"request_address": c.ClientIP(),
	})
	log.Debug("Processing email verification resend.")

	// ShouldBindJSON, not BindJSON: BindJSON writes a 400 itself on failure, which
	// would leak the difference between a malformed request and an accepted one.
	var req RequestBodyEmailVerifyResendPOST
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.JSON(200, gin.H{})
		return
	}

	sh := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := sh.CustomerEmailVerifyResend(c.Request.Context(), req.Email); err != nil {
		log.Debugf("Email verification resend failed. err: %v", err)
	}

	c.JSON(200, gin.H{})
}

// GetCustomerEmailVerify handles GET /auth/email-verify request.
// It serves a simple HTML page that auto-submits the verification token.
func GetCustomerEmailVerify(c *gin.Context) {
	token := c.Query("token")
	if !validVerifyToken.MatchString(token) {
		c.AbortWithStatus(400)
		return
	}

	html := fmt.Sprintf(emailVerifyHTML, token)
	c.Data(200, "text/html; charset=utf-8", []byte(html))
}

const emailVerifyHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Verify Email - VoIPBin</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; display: flex; justify-content: center; align-items: center; min-height: 100vh; }
  .container { background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); width: 100%%; max-width: 400px; text-align: center; }
  h1 { font-size: 24px; margin-bottom: 8px; color: #333; }
  p { color: #666; margin-bottom: 24px; font-size: 14px; }
  button { width: 100%%; padding: 12px; background: #4a90d9; color: white; border: none; border-radius: 4px; font-size: 16px; cursor: pointer; }
  button:hover { background: #357abd; }
  button:disabled { background: #ccc; cursor: not-allowed; }
  .message { padding: 12px; border-radius: 4px; margin-top: 16px; font-size: 14px; display: none; }
  .message.success { display: block; background: #e8f5e9; color: #2e7d32; border: 1px solid #c8e6c9; }
  .message.error { display: block; background: #ffebee; color: #c62828; border: 1px solid #ffcdd2; }
  #resend-box { margin-top: 24px; padding-top: 24px; border-top: 1px solid #eee; text-align: left; }
  #resend-box p { margin-bottom: 12px; }
  #resend-box input { width: 100%%; padding: 12px; margin-bottom: 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; }
</style>
</head>
<body>
<div class="container">
  <h1>Verify Your Email</h1>
  <p>Click the button below to verify your email address and activate your VoIPBin account.</p>
  <button id="verifyBtn" onclick="verify()">Verify Email</button>
  <div id="message" class="message"></div>
  <div id="resend-box" style="display:none">
    <p>Enter your email address and we will send a new verification link.</p>
    <input id="resend-email" type="email" placeholder="you@example.com" autocomplete="email">
    <button id="resend-btn" onclick="resend()">Send new link</button>
    <p id="resend-msg" class="message"></p>
  </div>
</div>
<script>
  var token = "%s";
  function verify() {
    var btn = document.getElementById('verifyBtn');
    var msgEl = document.getElementById('message');
    btn.disabled = true;
    btn.textContent = 'Verifying...';
    msgEl.className = 'message';

    fetch('/auth/email-verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: token })
    }).then(function(resp) {
      if (resp.ok) {
        msgEl.textContent = 'Email verified successfully! Check your inbox for a welcome email with instructions to set your password.';
        msgEl.className = 'message success';
        btn.style.display = 'none';
      } else {
        msgEl.textContent = 'This verification link is no longer valid.';
        msgEl.className = 'message error';
        btn.style.display = 'none';
        document.getElementById('resend-box').style.display = 'block';
      }
    }).catch(function() {
      msgEl.textContent = 'An error occurred. Please try again.';
      msgEl.className = 'message error';
      btn.disabled = false;
      btn.textContent = 'Verify Email';
    });
  }

  function resend() {
    var emailEl = document.getElementById('resend-email');
    var outEl = document.getElementById('resend-msg');
    var rbtn = document.getElementById('resend-btn');
    rbtn.disabled = true;
    fetch('/auth/email-verify-resend', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: emailEl.value })
    }).then(function() {
      // Always the same copy. The API deliberately returns 200 for unknown
      // addresses, and branching the message here would undo that.
      outEl.textContent = 'If an account exists for that address, a new verification link is on its way.';
      outEl.className = 'message success';
    }).catch(function() {
      outEl.textContent = 'If an account exists for that address, a new verification link is on its way.';
      outEl.className = 'message success';
    });
  }
</script>
</body>
</html>`
