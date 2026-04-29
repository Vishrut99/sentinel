package handler

import (
	"encoding/json"
	"net/http"

	"errors"

	"github.com/yourusername/incident-ticketing/internal/domain"
)

// WriteJSON writes a JSON response with status code.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(data)
}

// WriteError writes a standard API error response.
func WriteError(w http.ResponseWriter, status int, code, msg string) {
	WriteJSON(w, status, map[string]string{
		"error":   code,
		"message": msg,
	})
}

// WriteServiceError translates domain service errors into API responses.
func WriteServiceError(w http.ResponseWriter, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		status := http.StatusInternalServerError
		switch appErr.Code {
		case "BAD_REQUEST":
			status = http.StatusBadRequest
		case "UNAUTHORIZED":
			status = http.StatusUnauthorized
		case "FORBIDDEN":
			status = http.StatusForbidden
		case "NOT_FOUND":
			status = http.StatusNotFound
		case "CONFLICT":
			status = http.StatusConflict
		}
		WriteError(w, status, appErr.Code, appErr.Message)
		return
	}

	WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
}
