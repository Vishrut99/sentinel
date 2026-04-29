package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/repository"
)

type AuditService interface {
	GetByTicketID(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID, role string) ([]domain.AuditLog, error)
}

type auditService struct {
	auditRepo  repository.AuditRepository
	ticketRepo ticketAccessReader
}

func NewAuditService(auditRepo repository.AuditRepository, ticketRepo ticketAccessReader) AuditService {
	return &auditService{auditRepo: auditRepo, ticketRepo: ticketRepo}
}

func (s *auditService) GetByTicketID(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID, role string) ([]domain.AuditLog, error) {
	ticket, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("AuditService.GetByTicketID: %w", err)
	}
	if err := ensureTicketAccess(ticket, actorID, role); err != nil {
		return nil, fmt.Errorf("AuditService.GetByTicketID: %w", err)
	}

	logs, err := s.auditRepo.GetByTicketID(ctx, ticketID)
	if err != nil {
		return nil, fmt.Errorf("AuditService.GetByTicketID: %w", err)
	}
	return logs, nil
}
