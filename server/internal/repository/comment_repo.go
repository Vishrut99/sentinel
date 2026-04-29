package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
	"gorm.io/gorm"
)

type CommentRepository interface {
	Create(ctx context.Context, comment *domain.Comment) error
	ListByTicketID(ctx context.Context, ticketID uuid.UUID, includeInternal bool) ([]domain.Comment, error)
}

type commentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}
func (c *commentRepository) Create(ctx context.Context, comment *domain.Comment) error {

	if err := c.db.WithContext(ctx).Create(comment).Error; err != nil {
		return fmt.Errorf("CommentRepository.Create: %w", err)
	}
	return nil
}
func (c *commentRepository) ListByTicketID(ctx context.Context, ticketID uuid.UUID, includeInternal bool) ([]domain.Comment, error) {
	query := c.db.WithContext(ctx).Where("ticket_id = ?", ticketID)
	if !includeInternal {
		query = query.Where("is_internal = ?", false)
	}
	var comments []domain.Comment
	if err := query.Order("created_at asc").Preload("Author").Find(&comments).Error; err != nil {
		return nil, fmt.Errorf("CommentRepository.List: %w", err)
	}
	return comments, nil
}
