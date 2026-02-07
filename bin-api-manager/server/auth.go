package server

import (
	"fmt"
	"regexp"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var validResetToken = regexp.MustCompile(`^[0-9a-f]{64}$`)

func (h *server) PostAuthLogin(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAuthLogin",
		"request_address": c.ClientIP,
	})

	var req openapi_server.PostAuthLoginJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log = log.WithFields(logrus.Fields{
		"username": req.Username,
	})
	log.Debugf("Logging in. username: %s", req.Username)

	token, err := h.serviceHandler.AuthLogin(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		log.Debugf("Login failed. err: %v", err)
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("Created token string. token: %v", token)

	c.SetCookie("token", token, int(servicehandler.TokenExpiration.Seconds()), "/", "", false, true)
	res := openapi_server.AuthLoginResponse{
		Username: req.Username,
		Token:    token,
	}
	log.Debug("User successfully logged in.")

	c.JSON(200, res)
}

func (h *server) PostAuthPasswordForgot(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAuthPasswordForgot",
		"request_address": c.ClientIP,
	})

	var req openapi_server.PostAuthPasswordForgotJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log.Debugf("Processing password forgot. username: %s", req.Username)
	_ = h.serviceHandler.AuthPasswordForgot(c.Request.Context(), req.Username)

	c.JSON(200, gin.H{})
}

func (h *server) GetAuthPasswordReset(c *gin.Context, params openapi_server.GetAuthPasswordResetParams) {
	token := params.Token
	if !validResetToken.MatchString(token) {
		c.AbortWithStatus(400)
		return
	}

	html := fmt.Sprintf(passwordResetHTML, token)
	c.Data(200, "text/html; charset=utf-8", []byte(html))
}

func (h *server) PostAuthPasswordReset(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostAuthPasswordReset",
		"request_address": c.ClientIP,
	})
	log.Debug("Processing password reset.")

	var req openapi_server.PostAuthPasswordResetJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Warnf("Could not bind the request body. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	if err := h.serviceHandler.AuthPasswordReset(c.Request.Context(), req.Token, req.Password); err != nil {
		log.Debugf("Password reset failed. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	log.Debug("Password reset successful.")
	c.JSON(200, gin.H{})
}

const passwordResetHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Reset Password - VoIPBin</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; display: flex; justify-content: center; align-items: center; min-height: 100vh; }
  .container { background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); width: 100%%; max-width: 400px; }
  h1 { font-size: 24px; margin-bottom: 8px; color: #333; }
  p { color: #666; margin-bottom: 24px; font-size: 14px; }
  label { display: block; font-size: 14px; font-weight: 500; color: #333; margin-bottom: 6px; }
  input[type="password"] { width: 100%%; padding: 10px 12px; border: 1px solid #ddd; border-radius: 4px; font-size: 14px; margin-bottom: 16px; }
  input[type="password"]:focus { outline: none; border-color: #4a90d9; }
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
  <h1>Reset Password</h1>
  <p>Enter your new password below.</p>
  <form id="resetForm">
    <label for="password">New Password</label>
    <input type="password" id="password" name="password" required minlength="8" placeholder="Minimum 8 characters">
    <label for="confirmPassword">Confirm Password</label>
    <input type="password" id="confirmPassword" name="confirmPassword" required minlength="8" placeholder="Confirm your password">
    <button type="submit" id="submitBtn">Reset Password</button>
  </form>
  <div id="message" class="message"></div>
</div>
<script>
  var token = "%s";
  document.getElementById('resetForm').addEventListener('submit', function(e) {
    e.preventDefault();
    var password = document.getElementById('password').value;
    var confirm = document.getElementById('confirmPassword').value;
    var msgEl = document.getElementById('message');
    var btn = document.getElementById('submitBtn');

    msgEl.className = 'message';
    msgEl.style.display = 'none';

    if (password !== confirm) {
      msgEl.textContent = 'Passwords do not match.';
      msgEl.className = 'message error';
      return;
    }
    if (password.length < 8) {
      msgEl.textContent = 'Password must be at least 8 characters.';
      msgEl.className = 'message error';
      return;
    }

    btn.disabled = true;
    btn.textContent = 'Resetting...';

    fetch('/auth/password-reset', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: token, password: password })
    }).then(function(resp) {
      if (resp.ok) {
        msgEl.textContent = 'Password updated successfully. You can now log in with your new password.';
        msgEl.className = 'message success';
        document.getElementById('resetForm').style.display = 'none';
      } else {
        msgEl.textContent = 'Invalid or expired reset link. Please request a new one.';
        msgEl.className = 'message error';
        btn.disabled = false;
        btn.textContent = 'Reset Password';
      }
    }).catch(function() {
      msgEl.textContent = 'An error occurred. Please try again.';
      msgEl.className = 'message error';
      btn.disabled = false;
      btn.textContent = 'Reset Password';
    });
  });
</script>
</body>
</html>`
