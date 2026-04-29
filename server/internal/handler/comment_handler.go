package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/middleware"
	"github.com/yourusername/incident-ticketing/internal/service"
)

type CommentHandler struct {
	commentService service.CommentService
	validator      *validator.Validate
}

func NewCommentHandler(commentService service.CommentService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
		validator:      validator.New(),
	}
}

func (c *CommentHandler) RegisterRoutes(r chi.Router) {
	r.Route("/tickets/{id}/comments", func(r chi.Router) {
		r.Post("/", c.CreateComment)
		r.Get("/", c.ListComment)
	})
}

func (c *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateCommentReq
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON body")
		return
	}
	if err := c.validator.Struct(req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	ticketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid ticket ID")
		return
	}
	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	role, ok := middleware.RoleFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	com, err := c.commentService.CreateComment(r.Context(), ticketID, req, user, role)
	if err != nil {
		c.writeServiceError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, com)

}
func (c *CommentHandler) ListComment(w http.ResponseWriter, r *http.Request) {
	ticketID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid ticket ID")
		return
	}
	role, ok := middleware.RoleFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	comments, err := c.commentService.ListComments(r.Context(), ticketID, userID, role)
	if err != nil {
		c.writeServiceError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, comments)

}

func (c *CommentHandler) writeServiceError(w http.ResponseWriter, err error) {
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
