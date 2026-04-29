package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/service"
)

// AuthHandler handles auth HTTP endpoints.
type AuthHandler struct {
	authService service.AuthService
	validator   *validator.Validate
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validator:   validator.New(),
	}
}

// RegisterRoutes attaches auth routes to router.
func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/auth/bootstrap-admin", h.BootstrapAdmin)
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterReq
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	res, err := h.authService.Register(r.Context(), req)
	if err != nil {
		h.handleError(w, "Register", err)
		return
	}

	WriteJSON(w, http.StatusCreated, res)
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginReq
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	res, err := h.authService.Login(r.Context(), req)
	if err != nil {
		h.handleError(w, "Login", err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

func (h *AuthHandler) BootstrapAdmin(w http.ResponseWriter, r *http.Request) {
	var req domain.BootstrapAdminReq
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}

	if err := h.validator.Struct(req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	res, err := h.authService.BootstrapAdmin(r.Context(), req)
	if err != nil {
		h.handleError(w, "BootstrapAdmin", err)
		return
	}

	WriteJSON(w, http.StatusCreated, res)
}

func (h *AuthHandler) handleError(w http.ResponseWriter, method string, err error) {
	log.Printf("AuthHandler.%s: %v", method, err)

	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		WriteError(w, statusFromCode(appErr.Code), appErr.Code, appErr.Message)
		return
	}

	WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}

func statusFromCode(code string) int {
	switch code {
	case "BAD_REQUEST":
		return http.StatusBadRequest
	case "UNAUTHORIZED":
		return http.StatusUnauthorized
	case "FORBIDDEN":
		return http.StatusForbidden
	case "NOT_FOUND":
		return http.StatusNotFound
	case "CONFLICT":
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("AuthHandler.decodeJSON: %w", err)
	}
	if decoder.More() {
		return fmt.Errorf("AuthHandler.decodeJSON: multiple JSON values are not allowed")
	}
	return nil
}
