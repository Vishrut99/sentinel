package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/middleware"
	"github.com/yourusername/incident-ticketing/internal/service"
)

type AuditHandler struct {
	auditService service.AuditService
}

func NewAuditHandler(auditService service.AuditService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// RegisterRoutes registers the audit log route directly inside the tickets path grouping.
func (h *AuditHandler) RegisterRoutes(r chi.Router) {
	r.Get("/tickets/{id}/audit", h.GetTicketAudit)
}

func (h *AuditHandler) GetTicketAudit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	ticketID, err := uuid.Parse(idStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid ticket id")
		return
	}

	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	role, ok := middleware.RoleFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}

	logs, err := h.auditService.GetByTicketID(r.Context(), ticketID, actorID, role)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	responses := make([]domain.AuditLogRes, 0, len(logs))
	for _, l := range logs {
		var oldVal, newVal map[string]any

		if len(l.OldValue) > 0 && string(l.OldValue) != "null" {
			_ = json.Unmarshal(l.OldValue, &oldVal)
		}
		if len(l.NewValue) > 0 && string(l.NewValue) != "null" {
			_ = json.Unmarshal(l.NewValue, &newVal)
		}

		responses = append(responses, domain.AuditLogRes{
			ID:     l.ID,
			Action: l.Action,
			Actor: domain.UserSummary{
				ID:       l.Actor.ID,
				Email:    l.Actor.Email,
				FullName: l.Actor.FullName,
				Role:     l.Actor.Role,
			},
			OldValue:  oldVal,
			NewValue:  newVal,
			CreatedAt: l.CreatedAt,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]any{"data": responses})
}
