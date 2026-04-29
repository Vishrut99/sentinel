package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/service"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.UserListFilter{
		Role:   strings.TrimSpace(r.URL.Query().Get("role")),
		Search: strings.TrimSpace(r.URL.Query().Get("q")),
	}

	users, err := h.userService.ListUsers(r.Context(), filter)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{"data": users})
}

func (h *UserHandler) PromoteToAdmin(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || userID == uuid.Nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user id")
		return
	}

	res, err := h.userService.PromoteToAdmin(r.Context(), userID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"user":    res,
		"message": "user promoted to admin; re-login to get updated role claims",
	})
}

func (h *UserHandler) writeServiceError(w http.ResponseWriter, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		WriteError(w, statusFromCode(appErr.Code), appErr.Code, appErr.Message)
		return
	}

	WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
