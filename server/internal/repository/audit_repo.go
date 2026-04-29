package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
	"gorm.io/gorm"
)

type AuditRepository interface {
	GetByTicketID(ctx context.Context, ticketID uuid.UUID) ([]domain.AuditLog, error)
}

type auditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) GetByTicketID(ctx context.Context, ticketID uuid.UUID) ([]domain.AuditLog, error) {
	var logs []domain.AuditLog
	if err := r.db.WithContext(ctx).Preload("Actor").Where("ticket_id = ?", ticketID).Order("created_at desc").Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("auditRepository.GetByTicketID: %w", err)
	}
	return logs, nil
}
