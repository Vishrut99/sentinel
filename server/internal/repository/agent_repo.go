package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
	"gorm.io/gorm"
)

type AgentRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Agent, error)
	List(ctx context.Context) ([]domain.Agent, error)
	UpdateCapacity(ctx context.Context, id uuid.UUID, isAvailable *bool, maxTickets *int, skills *domain.SkillScoreList) error
}

type agentRepository struct {
	db *gorm.DB
}

func NewAgentRepository(db *gorm.DB) AgentRepository {
	return &agentRepository{db: db}
}

func (r *agentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	var agent domain.Agent
	if err := r.db.WithContext(ctx).Preload("User").Where("id = ?", id).First(&agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("agentRepository.GetByID: %w", err)
	}
	return &agent, nil
}

func (r *agentRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Agent, error) {
	var agent domain.Agent
	if err := r.db.WithContext(ctx).Preload("User").Where("user_id = ?", userID).First(&agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("agentRepository.GetByUserID: %w", err)
	}
	return &agent, nil
}

func (r *agentRepository) List(ctx context.Context) ([]domain.Agent, error) {
	var agents []domain.Agent
	if err := r.db.WithContext(ctx).Preload("User").Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("agentRepository.List: %w", err)
	}
	return agents, nil
}

func (r *agentRepository) UpdateCapacity(ctx context.Context, id uuid.UUID, isAvailable *bool, maxTickets *int, skills *domain.SkillScoreList) error {
	updates := make(map[string]any)
	if isAvailable != nil {
		updates["is_available"] = *isAvailable
	}
	if maxTickets != nil {
		updates["max_tickets"] = *maxTickets
	}
	if skills != nil {
		updates["skills"] = *skills
	}

	result := r.db.WithContext(ctx).Model(&domain.Agent{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("agentRepository.UpdateCapacity: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
