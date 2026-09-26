package http

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"workshop/internal/domain"
	"workshop/internal/usecase"
)

// signInRequest is the credential payload the login screen sends.
type signInRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// sessionResponse is what a successful sign in returns. It never carries the
// password hash.
type sessionResponse struct {
	Token                  string `json:"token"`
	ExpiresAt              string `json:"expiresAt"`
	UserID                 string `json:"userId"`
	Username               string `json:"username"`
	FullName               string `json:"fullName"`
	Role                   string `json:"role"`
	RequiresPasswordChange bool   `json:"requiresPasswordChange"`
}

// AuthHandler exposes the sign in operation.
type AuthHandler struct {
	authenticate usecase.AuthenticateUser
	limiter      *LoginRateLimiter
}

// NewAuthHandler wires the authentication handler.
func NewAuthHandler(authenticate usecase.AuthenticateUser, limiter *LoginRateLimiter) AuthHandler {
	if limiter == nil {
		limiter = NewLoginRateLimiter()
	}
	return AuthHandler{
		authenticate: authenticate,
		limiter:      limiter,
	}
}

// SignIn verifies the credentials and returns a session token.
func (h AuthHandler) SignIn(writer http.ResponseWriter, request *http.Request) {
	var payload signInRequest
	if err := decode(writer, request, &payload); err != nil {
		failure(writer, err)
		return
	}

	clientIP := h.limiter.ExtractClientIP(request, nil)
	allowed, retryAfter := h.limiter.AllowAttempt(clientIP, payload.Username)
	if !allowed {
		seconds := int(math.Ceil(retryAfter.Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		writer.Header().Set("Retry-After", strconv.Itoa(seconds))
		respond(writer, http.StatusTooManyRequests, errorPayload{
			Code:    "rate_limit_exceeded",
			Message: "Demasiados intentos fallidos. Intente nuevamente mas tarde.",
		})
		return
	}

	session, err := h.authenticate.Execute(request.Context(), payload.Username, payload.Password)
	if err != nil {
		h.limiter.RecordFailure(clientIP, payload.Username)
		if errors.Is(err, domain.ErrAccountInactive) {
			failure(writer, domain.ErrAccountInactive)
			return
		}
		failure(writer, domain.ErrInvalidCredentials)
		return
	}

	h.limiter.Reset(clientIP, payload.Username)
	respond(writer, http.StatusOK, sessionResponse{
		Token:                  session.Token,
		ExpiresAt:              formatTime(session.ExpiresAt),
		UserID:                 session.UserID,
		Username:               session.Username,
		FullName:               session.FullName,
		Role:                   string(session.Role),
		RequiresPasswordChange: session.RequiresPasswordChange,
	})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// ChangePassword allows an authenticated user to change their password and clear requires_password_change.
func (h AuthHandler) ChangePassword(writer http.ResponseWriter, request *http.Request) {
	identity, err := callerFrom(request.Context())
	if err != nil {
		failure(writer, err)
		return
	}
	var payload changePasswordRequest
	if err := decode(writer, request, &payload); err != nil {
		failure(writer, err)
		return
	}
	if err := h.authenticate.ChangePassword(request.Context(), identity.UserID, payload.CurrentPassword, payload.NewPassword); err != nil {
		failure(writer, err)
		return
	}
	respond(writer, http.StatusOK, map[string]string{"message": "contraseña actualizada exitosamente"})
}
