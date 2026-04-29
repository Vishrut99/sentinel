package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/repository"
)

type AgentService interface {
	Register(ctx context.Context, req domain.RegisterAgentReq, actorID uuid.UUID) (*domain.AgentSummary, error)
	List(ctx context.Context) ([]domain.AgentSummary, error)
	UpdateCapacity(ctx context.Context, agentID uuid.UUID, req domain.UpdateAgentCapacityReq) (*domain.AgentSummary, error)
}

type agentService struct {
	agentrepo repository.AgentRepository
	db        *gorm.DB
}

func NewAgentService(agentRepo repository.AgentRepository, db *gorm.DB) AgentService {
	return &agentService{
		agentrepo: agentRepo,
		db:        db,
	}
}

func (s *agentService) Register(ctx context.Context, req domain.RegisterAgentReq, actorID uuid.UUID) (*domain.AgentSummary, error) {
	department := strings.TrimSpace(req.Department)
	if department == "" {
		return nil, fmt.Errorf("AgentService.Register: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("department is required")))
	}

	maxTickets := 10
	if req.MaxTickets != nil {
		maxTickets = *req.MaxTickets
	}

	result := s.db.WithContext(ctx).Exec("CALL sp_register_agent($1, $2, $3, $4)", req.UserID, department, maxTickets, actorID)
	if result.Error != nil {
		return nil, fmt.Errorf("AgentService.Register: %w", mapRegisterAgentProcedureError(result.Error))
	}

	agent, err := s.agentrepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("AgentService.Register GetByUserID: %w", err)
	}

	skills, err := sanitizeSkillScores(req.Skills)
	if err != nil {
		return nil, fmt.Errorf("AgentService.Register: %w", domain.ErrBadRequest.WithErr(err))
	}
	if len(skills) == 0 {
		skills = defaultAgentSkillsForDepartment(department)
	}
	if err := s.db.WithContext(ctx).Model(&domain.Agent{}).Where("id = ?", agent.ID).Update("skills", skills).Error; err != nil {
		return nil, fmt.Errorf("AgentService.Register: %w", err)
	}

	agent.Skills = skills
	activeCounts, err := fetchActiveTicketCounts(ctx, s.db, []uuid.UUID{agent.ID})
	if err != nil {
		return nil, fmt.Errorf("AgentService.Register: %w", err)
	}

	return buildAgentSummary(agent, activeCounts[agent.ID]), nil
}

func mapRegisterAgentProcedureError(err error) error {
	if err == nil {
		return nil
	}

	lowerErr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lowerErr, "user not found"):
		return &domain.AppError{Code: domain.ErrNotFound.Code, Message: "user not found", Err: err}
	case strings.Contains(lowerErr, "already an agent or admin"):
		return &domain.AppError{Code: domain.ErrConflict.Code, Message: "user is already an agent or admin", Err: err}
	case strings.Contains(lowerErr, "max_tickets must be at least 1"):
		return &domain.AppError{Code: domain.ErrBadRequest.Code, Message: "max_tickets must be at least 1", Err: err}
	default:
		return err
	}
}

func (s *agentService) List(ctx context.Context) ([]domain.AgentSummary, error) {
	agents, err := s.agentrepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("AgentService.List: %w", err)
	}

	agentIDs := make([]uuid.UUID, 0, len(agents))
	for _, agent := range agents {
		agentIDs = append(agentIDs, agent.ID)
	}

	activeCounts, err := fetchActiveTicketCounts(ctx, s.db, agentIDs)
	if err != nil {
		return nil, fmt.Errorf("AgentService.List: %w", err)
	}

	summaries := make([]domain.AgentSummary, 0, len(agents))
	for _, a := range agents {
		agent := a
		summaries = append(summaries, *buildAgentSummary(&agent, activeCounts[agent.ID]))
	}
	return summaries, nil
}

func (s *agentService) UpdateCapacity(ctx context.Context, agentID uuid.UUID, req domain.UpdateAgentCapacityReq) (*domain.AgentSummary, error) {
	if req.IsAvailable == nil && req.MaxTickets == nil && req.Skills == nil {
		return nil, fmt.Errorf("AgentService.UpdateCapacity: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("at least one capacity field is required")))
	}

	var skills *domain.SkillScoreList
	if req.Skills != nil {
		sanitized, err := sanitizeSkillScores(*req.Skills)
		if err != nil {
			return nil, fmt.Errorf("AgentService.UpdateCapacity: %w", domain.ErrBadRequest.WithErr(err))
		}
		skills = &sanitized
	}

	if err := s.agentrepo.UpdateCapacity(ctx, agentID, req.IsAvailable, req.MaxTickets, skills); err != nil {
		return nil, fmt.Errorf("AgentService.UpdateCapacity: %w", err)
	}

	agent, err := s.agentrepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("AgentService.UpdateCapacity: %w", err)
	}

	activeCounts, err := fetchActiveTicketCounts(ctx, s.db, []uuid.UUID{agent.ID})
	if err != nil {
		return nil, fmt.Errorf("AgentService.UpdateCapacity: %w", err)
	}

	return buildAgentSummary(agent, activeCounts[agent.ID]), nil
}

func buildAgentSummary(agent *domain.Agent, activeTickets int) *domain.AgentSummary {
	department := ""
	if agent.Department != nil {
		department = *agent.Department
	}

	utilizationPercent := 0
	if agent.MaxTickets > 0 {
		utilizationPercent = int(math.Round((float64(activeTickets) / float64(agent.MaxTickets)) * 100))
	}

	return &domain.AgentSummary{
		ID:                 agent.ID,
		FullName:           agent.User.FullName,
		Department:         department,
		IsAvailable:        agent.IsAvailable,
		MaxTickets:         agent.MaxTickets,
		Skills:             agent.Skills,
		ActiveTickets:      activeTickets,
		UtilizationPercent: utilizationPercent,
	}
}
