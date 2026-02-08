package server

import (
	"fmt"
	"regexp"

	"monorepo/bin-api-manager/gens/openapi_server"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var validVerifyToken = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (h *server) PostAuthSignup(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAuthSignup",
		"request_address": c.ClientIP,
	})

	var req openapi_server.PostAuthSignupJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log = log.WithFields(logrus.Fields{
		"email": req.Email,
	})
	log.Debugf("Processing customer signup. email: %s", req.Email)

	webhookMethod := ""
	if req.WebhookMethod != nil {
		webhookMethod = string(*req.WebhookMethod)
	}
	webhookURI := ""
	if req.WebhookUri != nil {
		webhookURI = *req.WebhookUri
	}
	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	detail := ""
	if req.Detail != nil {
		detail = *req.Detail
	}
	phoneNumber := ""
	if req.PhoneNumber != nil {
		phoneNumber = *req.PhoneNumber
	}
	address := ""
	if req.Address != nil {
		address = *req.Address
	}

	res, err := h.serviceHandler.CustomerSignup(
		c.Request.Context(),
		name,
		detail,
		req.Email,
		phoneNumber,
		address,
		cscustomer.WebhookMethod(webhookMethod),
		webhookURI,
	)
	if err != nil {
		log.Debugf("Customer signup failed. err: %v", err)
		// Return 200 with empty body to prevent email enumeration
		c.JSON(200, gin.H{})
		return
	}

	c.JSON(200, res)
}

func (h *server) GetAuthEmailVerify(c *gin.Context, params openapi_server.GetAuthEmailVerifyParams) {
	token := params.Token
	if !validVerifyToken.MatchString(token) {
		c.AbortWithStatus(400)
		return
	}

	html := fmt.Sprintf(emailVerifyHTML, token)
	c.Data(200, "text/html; charset=utf-8", []byte(html))
}

func (h *server) PostAuthEmailVerify(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAuthEmailVerify",
		"request_address": c.ClientIP,
	})
	log.Debug("Processing email verification.")

	var req openapi_server.PostAuthEmailVerifyJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.CustomerEmailVerify(c.Request.Context(), req.Token)
	if err != nil {
		log.Debugf("Email verification failed. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
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
</style>
</head>
<body>
<div class="container">
  <h1>Verify Your Email</h1>
  <p>Click the button below to verify your email address and activate your VoIPBin account.</p>
  <button id="verifyBtn" onclick="verify()">Verify Email</button>
  <div id="message" class="message"></div>
</div>
<script>
  var token = "%s";
  function verify() {
    var btn = document.getElementById('verifyBtn');
    var msgEl = document.getElementById('message');
    btn.disabled = true;
    btn.textContent = 'Verifying...';
    msgEl.className = 'message';
    msgEl.style.display = 'none';

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
        msgEl.textContent = 'Invalid or expired verification link. Please sign up again.';
        msgEl.className = 'message error';
        btn.disabled = false;
        btn.textContent = 'Verify Email';
      }
    }).catch(function() {
      msgEl.textContent = 'An error occurred. Please try again.';
      msgEl.className = 'message error';
      btn.disabled = false;
      btn.textContent = 'Verify Email';
    });
  }
</script>
</body>
</html>`
