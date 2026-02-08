package handlers

import (
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"subvault/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService    service.AuthServiceInterface
	sessionService *service.SessionService
	emailService   service.EmailServiceInterface
	notifConfig    service.NotificationConfigServiceInterface
}

func NewAuthHandler(authService service.AuthServiceInterface, sessionService *service.SessionService, emailService service.EmailServiceInterface, notifConfig service.NotificationConfigServiceInterface) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		sessionService: sessionService,
		emailService:   emailService,
		notifConfig:    notifConfig,
	}
}

// isValidRedirect validates that a redirect URL is safe (relative URL only)
func isValidRedirect(redirect string) bool {
	if len(redirect) > 2048 {
		return false
	}
	if strings.HasPrefix(redirect, "/") && !strings.HasPrefix(redirect, "//") {
		return true
	}
	return false
}

// ShowLoginPage displays the login page
func (h *AuthHandler) ShowLoginPage(c *gin.Context) {
	if h.sessionService.IsAuthenticated(c.Request) {
		c.Redirect(http.StatusFound, "/")
		return
	}

	redirect := c.Query("redirect")
	if redirect == "" || !isValidRedirect(redirect) {
		redirect = "/"
	}

	data := baseTemplateData(c)
	mergeTemplateData(data, gin.H{
		"Redirect": redirect,
		"Error":    c.Query("error"),
	})
	c.HTML(http.StatusOK, "login.html", data)
}

// Login handles login form submission
func (h *AuthHandler) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	rememberMe := c.PostForm("remember_me") == "on"
	redirect := c.PostForm("redirect")

	if redirect == "" || !isValidRedirect(redirect) {
		redirect = "/"
	}

	storedUsername, err := h.authService.GetAuthUsername()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login-error.html", gin.H{
			"Error": tr(c, "auth_error_system", "Authentication system error"),
		})
		return
	}

	validUsername := subtle.ConstantTimeCompare([]byte(storedUsername), []byte(username)) == 1

	var validPassword bool
	if err := h.authService.ValidatePassword(password); err == nil {
		validPassword = true
	}

	if !validUsername || !validPassword {
		c.HTML(http.StatusUnauthorized, "login-error.html", gin.H{
			"Error": tr(c, "auth_error_invalid_credentials", "Invalid username or password"),
		})
		return
	}

	if err := h.sessionService.CreateSession(c.Writer, c.Request, rememberMe); err != nil {
		c.HTML(http.StatusInternalServerError, "login-error.html", gin.H{
			"Error": tr(c, "auth_error_session", "Failed to create session"),
		})
		return
	}

	c.Header("HX-Redirect", redirect)
	c.Status(http.StatusOK)
}

// Logout handles logout
func (h *AuthHandler) Logout(c *gin.Context) {
	if err := h.sessionService.DestroySession(c.Writer, c.Request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}
	c.Redirect(http.StatusFound, "/login")
}

// ShowForgotPasswordPage displays the forgot password page
func (h *AuthHandler) ShowForgotPasswordPage(c *gin.Context) {
	data := baseTemplateData(c)
	c.HTML(http.StatusOK, "forgot-password.html", data)
}

// ForgotPassword handles forgot password request
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	token, err := h.authService.GenerateResetToken()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "forgot-password-error.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": tr(c, "auth_error_generate_token", "Failed to generate reset token"),
		}))
		return
	}

	_, err = h.notifConfig.GetSMTPConfig()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "forgot-password-error.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": tr(c, "auth_error_email_not_configured", "Email is not configured. Please contact administrator."),
		}))
		return
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	resetURL := fmt.Sprintf("%s://%s/reset-password?token=%s", scheme, c.Request.Host, url.QueryEscape(token))

	subject := "SubVault Password Reset"
	body := fmt.Sprintf(`
		<h2>Password Reset Request</h2>
		<p>You have requested to reset your SubVault password.</p>
		<p>Click the link below to reset your password:</p>
		<p><a href="%s">Reset Password</a></p>
		<p>This link will expire in 1 hour.</p>
		<p>If you did not request this reset, please ignore this email.</p>
	`, resetURL)

	err = h.emailService.SendEmail(subject, body)
	if err != nil {
		slog.Error("failed to send reset email", "error", err)
		c.HTML(http.StatusInternalServerError, "forgot-password-error.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": tr(c, "error_something_wrong", "Something went wrong"),
		}))
		return
	}

	c.HTML(http.StatusOK, "forgot-password-success.html", mergeTemplateData(baseTemplateData(c), gin.H{
		"Message": tr(c, "auth_success_reset_sent", "Password reset link has been sent to your email"),
	}))
}

// ShowResetPasswordPage displays the reset password page
func (h *AuthHandler) ShowResetPasswordPage(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.HTML(http.StatusBadRequest, "reset-password.html", gin.H{"Error": "Invalid reset token"})
		return
	}

	if err := h.authService.ValidateResetToken(token); err != nil {
		c.HTML(http.StatusBadRequest, "reset-password.html", gin.H{"Error": "Invalid or expired reset token"})
		return
	}

	data := baseTemplateData(c)
	mergeTemplateData(data, gin.H{"Token": token})
	c.HTML(http.StatusOK, "reset-password.html", data)
}

// ResetPassword handles password reset
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	token := c.PostForm("token")
	newPassword := c.PostForm("new_password")
	confirmPassword := c.PostForm("confirm_password")

	if len(newPassword) < 8 {
		c.HTML(http.StatusBadRequest, "reset-password-error.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": tr(c, "auth_error_password_short", "Password must be at least 8 characters long"),
		}))
		return
	}

	if newPassword != confirmPassword {
		c.HTML(http.StatusBadRequest, "reset-password-error.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": tr(c, "auth_error_password_mismatch", ErrPasswordsDoNotMatch),
		}))
		return
	}

	if err := h.authService.ValidateResetToken(token); err != nil {
		c.HTML(http.StatusBadRequest, "reset-password-error.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": tr(c, "auth_error_invalid_token", "Invalid or expired reset token"),
		}))
		return
	}

	if err := h.authService.SetAuthPassword(newPassword); err != nil {
		c.HTML(http.StatusInternalServerError, "reset-password-error.html", mergeTemplateData(baseTemplateData(c), gin.H{
			"Error": tr(c, "auth_error_update_password", "Failed to update password"),
		}))
		return
	}

	h.authService.ClearResetToken()

	c.HTML(http.StatusOK, "reset-password-success.html", mergeTemplateData(baseTemplateData(c), gin.H{
		"Message": tr(c, "auth_success_password_reset", "Password reset successfully. You can now login with your new password."),
	}))
}
