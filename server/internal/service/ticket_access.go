package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
)

type ticketAccessReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Ticket, error)
}

func ensureTicketAccess(ticket *domain.Ticket, actorID uuid.UUID, role string) error {
	if ticket == nil {
		return domain.ErrNotFound
	}

	switch role {
	case "admin":
		return nil
	case "agent":
		if ticket.Assignee == nil || ticket.Assignee.UserID != actorID {
			return domain.ErrForbidden
		}
		return nil
	case "user":
		if ticket.CreatedBy != actorID {
			return domain.ErrForbidden
		}
		return nil
	default:
		return domain.ErrForbidden
	}
}
