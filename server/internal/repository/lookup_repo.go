package repository

import (
	"context"
	"fmt"

	"github.com/yourusername/incident-ticketing/internal/domain"
	"gorm.io/gorm"
)

type LookupRepository interface {
	ListCategories(ctx context.Context) ([]domain.Category, error)
	ListPriorities(ctx context.Context) ([]domain.Priority, error)
	ListStatuses(ctx context.Context) ([]domain.TicketStatus, error)
}

type lookupRepository struct {
	db *gorm.DB
}

func NewLookupRepository(db *gorm.DB) LookupRepository {
	return &lookupRepository{db: db}
}

func (r *lookupRepository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	var categories []domain.Category
	if err := r.db.WithContext(ctx).Order("name asc").Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("lookupRepository.ListCategories: %w", err)
	}
	return categories, nil
}

func (r *lookupRepository) ListPriorities(ctx context.Context) ([]domain.Priority, error) {
	var priorities []domain.Priority
	if err := r.db.WithContext(ctx).Order("id asc").Find(&priorities).Error; err != nil {
		return nil, fmt.Errorf("lookupRepository.ListPriorities: %w", err)
	}
	return priorities, nil
}

func (r *lookupRepository) ListStatuses(ctx context.Context) ([]domain.TicketStatus, error) {
	var statuses []domain.TicketStatus
	if err := r.db.WithContext(ctx).Order("id asc").Find(&statuses).Error; err != nil {
		return nil, fmt.Errorf("lookupRepository.ListStatuses: %w", err)
	}
	return statuses, nil
}
