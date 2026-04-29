package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/cache"
	"github.com/yourusername/incident-ticketing/internal/domain"
	"github.com/yourusername/incident-ticketing/internal/middleware"
	"github.com/yourusername/incident-ticketing/internal/repository"
	"gorm.io/gorm"
)

type TicketService interface {
	CreateTicket(ctx context.Context, req domain.CreateTicketReq, userId uuid.UUID, role string) (*domain.Ticket, error)
	GetTicket(ctx context.Context, id uuid.UUID, userid uuid.UUID, role string) (*domain.Ticket, error)
	ListTicket(ctx context.Context, filter domain.TicketListFilter, userid uuid.UUID, role string) ([]domain.Ticket, int64, error)
	UpdateTicket(ctx context.Context, id uuid.UUID, req domain.UpdateTicketReq, actorID uuid.UUID, role string) (*domain.Ticket, error)
	ChangeStatus(ctx context.Context, id uuid.UUID, statusCode string, actorID uuid.UUID, role string) error
	AssignTicket(ctx context.Context, ticketID uuid.UUID, agentID uuid.UUID, actorID uuid.UUID) error
	LinkToProblem(ctx context.Context, ticketID uuid.UUID, req domain.LinkProblemReq, actorID uuid.UUID, role string) error
}

type ticketService struct {
	ticketRepo      repository.TicketRepository
	db              *gorm.DB
	redisCache      *cache.RedisCache
	aiTriageService AITriageService
}

func NewTicketService(tickerRepo repository.TicketRepository, db *gorm.DB, redisCache *cache.RedisCache, aiTriageService AITriageService) TicketService {
	return &ticketService{
		ticketRepo:      tickerRepo,
		db:              db,
		redisCache:      redisCache,
		aiTriageService: aiTriageService,
	}
}

func (s *ticketService) CreateTicket(ctx context.Context, req domain.CreateTicketReq, userId uuid.UUID, role string) (*domain.Ticket, error) {
	if req.IsProblem && !middleware.HasPermission(role, middleware.PermCreateProblemTicket) {
		return nil, fmt.Errorf("TicketService.CreateTicket: %w", domain.ErrForbidden)
	}
	if req.ParentID != nil {
		if *req.ParentID == uuid.Nil {
			return nil, fmt.Errorf("TicketService.CreateTicket: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("parent_id is invalid")))
		}
		if req.IsProblem {
			return nil, fmt.Errorf("TicketService.CreateTicket: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("problem ticket cannot have parent_id")))
		}
		parentTicket, err := s.ticketRepo.GetByID(ctx, *req.ParentID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, fmt.Errorf("TicketService.CreateTicket: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("parent problem ticket not found")))
			}
			return nil, fmt.Errorf("TicketService.CreateTicket: %w", err)
		}
		if !parentTicket.IsProblem {
			return nil, fmt.Errorf("TicketService.CreateTicket: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("parent_id must reference a problem ticket")))
		}
	}

	priority, err := s.ticketRepo.GetPriorityByName(ctx, req.Priority)
	if err != nil {
		return nil, fmt.Errorf("TicketService.CreateTicket: %w", err)
	}

	status, err := s.ticketRepo.GetStatusByName(ctx, "open")
	if err != nil {
		return nil, fmt.Errorf("TicketService.CreateTicket: %w", err)
	}

	var aiInsights []byte
	if s.aiTriageService != nil {
		aiInsights, err = s.aiTriageService.AnalyzeTicket(ctx, req.Title, req.Description)
		if err != nil {
			log.Printf("TicketService.CreateTicket AI triage failed: %v", err)
			fallback := buildAITriageFailurePayload(err)
			if fallback != nil {
				aiInsights = fallback
			}
		}
	}

	requiredSkills := resolveRequiredSkills(req, aiInsights)

	ticket := &domain.Ticket{
		ID:             uuid.New(),
		StatusID:       status.ID,
		CreatedBy:      userId,
		Title:          req.Title,
		Description:    &req.Description,
		AIInsights:     domain.JSONPayload(aiInsights),
		RequiredSkills: requiredSkills,
		PriorityID:     priority.ID,
		CategoryID:     req.CategoryID,
		IsProblem:      req.IsProblem,
		ParentID:       req.ParentID,
	}
	if err := s.ticketRepo.Create(ctx, ticket); err != nil {
		return nil, fmt.Errorf("TicketService.CreateTicket: %w", err)
	}

	createdTicket, err := s.ticketRepo.GetByID(ctx, ticket.ID)
	if err != nil {
		return nil, fmt.Errorf("TicketService.CreateTicket: %w", err)
	}

	if autoAssignErr := s.tryAutoAssignTicket(ctx, createdTicket, userId); autoAssignErr != nil {
		log.Printf("TicketService.CreateTicket auto-assignment skipped: %v", autoAssignErr)
	}

	createdTicket, err = s.ticketRepo.GetByID(ctx, ticket.ID)
	if err != nil {
		return nil, fmt.Errorf("TicketService.CreateTicket: %w", err)
	}
	s.invalidateDashboardCache(ctx)
	return createdTicket, nil
}

func (s *ticketService) LinkToProblem(ctx context.Context, ticketID uuid.UUID, req domain.LinkProblemReq, actorID uuid.UUID, role string) error {
	if !middleware.HasPermission(role, middleware.PermLinkProblem) {
		return fmt.Errorf("TicketService.LinkToProblem: %w", domain.ErrForbidden)
	}
	if ticketID == req.ParentID {
		return fmt.Errorf("TicketService.LinkToProblem: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("ticket_id and parent_id cannot be same")))
	}

	childTicket, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("TicketService.LinkToProblem: %w", err)
	}
	if err := ensureTicketAccess(childTicket, actorID, role); err != nil {
		return fmt.Errorf("TicketService.LinkToProblem: %w", err)
	}
	if childTicket.IsProblem {
		return fmt.Errorf("TicketService.LinkToProblem: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("problem ticket cannot be linked as child")))
	}

	parentTicket, err := s.ticketRepo.GetByID(ctx, req.ParentID)
	if err != nil {
		return fmt.Errorf("TicketService.LinkToProblem: %w", err)
	}
	if !parentTicket.IsProblem {
		return fmt.Errorf("TicketService.LinkToProblem: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("parent_id must reference a problem ticket")))
	}

	if err := s.ticketRepo.SetParent(ctx, ticketID, req.ParentID); err != nil {
		return fmt.Errorf("TicketService.LinkToProblem: %w", err)
	}

	s.invalidateDashboardCache(ctx)
	return nil
}

func (s *ticketService) GetTicket(ctx context.Context, id uuid.UUID, userid uuid.UUID, role string) (*domain.Ticket, error) {
	ticket, err := s.ticketRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("TicketService.GetTicket: %w", err)
	}
	if err := ensureTicketAccess(ticket, userid, role); err != nil {
		return nil, err
	}
	s.hydrateDerivedTicketMetadata(ticket)
	return ticket, nil
}

func (s *ticketService) ListTicket(ctx context.Context, filter domain.TicketListFilter, userid uuid.UUID, role string) ([]domain.Ticket, int64, error) {

	if middleware.HasPermission(role, middleware.PermViewAllTickets) {
		// admin: no filter, sees all tickets
	} else if middleware.HasPermission(role, middleware.PermViewAssignedTickets) {
		filter.AssignedUserID = userid
	} else {
		filter.CreatedBy = userid
	}
	tickets, total, err := s.ticketRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("TicketService.ListTicket: %w", err)
	}
	for i := range tickets {
		s.hydrateDerivedTicketMetadata(&tickets[i])
	}
	return tickets, total, nil
}

func (s *ticketService) UpdateTicket(ctx context.Context, id uuid.UUID, req domain.UpdateTicketReq, actorID uuid.UUID, role string) (*domain.Ticket, error) {
	ticket, err := s.ticketRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("TicketService.UpdateTicket: %w", err)
	}
	if err := ensureTicketAccess(ticket, actorID, role); err != nil {
		return nil, fmt.Errorf("TicketService.UpdateTicket: %w", err)
	}

	updates := make(map[string]any)
	oldValues := make(map[string]any)
	newValues := make(map[string]any)

	currentDescription := ""
	if ticket.Description != nil {
		currentDescription = *ticket.Description
	}

	if req.Title != nil {
		nextTitle := strings.TrimSpace(*req.Title)
		if len(nextTitle) < 5 {
			return nil, fmt.Errorf("TicketService.UpdateTicket: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("title must be at least 5 characters")))
		}
		if nextTitle != ticket.Title {
			updates["title"] = nextTitle
			oldValues["title"] = ticket.Title
			newValues["title"] = nextTitle
		}
	}

	if req.Description != nil {
		nextDescription := strings.TrimSpace(*req.Description)
		if nextDescription != currentDescription {
			if nextDescription == "" {
				updates["description"] = nil
			} else {
				updates["description"] = nextDescription
			}
			oldValues["description"] = currentDescription
			newValues["description"] = nextDescription
		}
	}

	effectivePriority := ticket.Priority
	updatedCategoryName := ticket.Category.Name
	priorityChanged := false
	categoryChanged := false

	if req.Priority != nil {
		nextPriorityName := strings.TrimSpace(*req.Priority)
		priority, err := s.ticketRepo.GetPriorityByName(ctx, nextPriorityName)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, fmt.Errorf("TicketService.UpdateTicket: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("priority not found")))
			}
			return nil, fmt.Errorf("TicketService.UpdateTicket: %w", err)
		}
		if priority.ID != ticket.PriorityID {
			priorityChanged = true
			effectivePriority = *priority
			updates["priority_id"] = priority.ID
			oldValues["priority"] = ticket.Priority.Name
			newValues["priority"] = priority.Name
		}
	}

	if req.CategoryID != nil {
		if *req.CategoryID == uuid.Nil {
			return nil, fmt.Errorf("TicketService.UpdateTicket: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("category_id is invalid")))
		}

		var nextCategory domain.Category
		if err := s.db.WithContext(ctx).Where("id = ?", *req.CategoryID).First(&nextCategory).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("TicketService.UpdateTicket: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("category not found")))
			}
			return nil, fmt.Errorf("TicketService.UpdateTicket: %w", err)
		}

		if nextCategory.ID != ticket.CategoryID {
			categoryChanged = true
			updatedCategoryName = nextCategory.Name
			updates["category_id"] = nextCategory.ID
			oldValues["category"] = ticket.Category.Name
			newValues["category"] = nextCategory.Name
		}
	}

	classificationChanged := priorityChanged || categoryChanged
	changeReason := ""
	if req.ChangeReason != nil {
		changeReason = strings.TrimSpace(*req.ChangeReason)
	}
	if classificationChanged && changeReason == "" {
		return nil, fmt.Errorf("TicketService.UpdateTicket: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("change_reason is required when priority or category changes")))
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("TicketService.UpdateTicket: %w", domain.ErrBadRequest.WithErr(fmt.Errorf("at least one field must change")))
	}

	if classificationChanged && !isTerminalTicketStatus(ticket.Status.Name) {
		newDueAt := time.Now().Add(time.Duration(effectivePriority.SLAResolveHrs) * time.Hour)
		updates["due_at"] = newDueAt
		updates["sla_breached"] = false
		oldValues["sla_breached"] = ticket.SLABreached
		newValues["sla_breached"] = false
		if ticket.DueAt != nil {
			oldValues["due_at"] = ticket.DueAt.Format(time.RFC3339)
		}
		newValues["due_at"] = newDueAt.Format(time.RFC3339)
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.Ticket{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return fmt.Errorf("TicketService.UpdateTicket update: %w", err)
		}

		oldPayload, err := json.Marshal(oldValues)
		if err != nil {
			return fmt.Errorf("TicketService.UpdateTicket marshal old values: %w", err)
		}
		newPayload, err := json.Marshal(newValues)
		if err != nil {
			return fmt.Errorf("TicketService.UpdateTicket marshal new values: %w", err)
		}

		auditLog := &domain.AuditLog{
			ID:        uuid.New(),
			TicketID:  id,
			ActorID:   actorID,
			Action:    "ticket_updated",
			OldValue:  domain.JSONPayload(oldPayload),
			NewValue:  domain.JSONPayload(newPayload),
			CreatedAt: time.Now(),
		}
		if err := tx.Create(auditLog).Error; err != nil {
			return fmt.Errorf("TicketService.UpdateTicket audit: %w", err)
		}

		if classificationChanged {
			comment := &domain.Comment{
				ID:         uuid.New(),
				TicketID:   id,
				AuthorID:   actorID,
				Body:       buildClassificationChangeComment(ticket, effectivePriority.Name, updatedCategoryName, changeReason, priorityChanged, categoryChanged),
				IsInternal: true,
				CreatedAt:  time.Now(),
			}
			if err := tx.Create(comment).Error; err != nil {
				return fmt.Errorf("TicketService.UpdateTicket comment: %w", err)
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("TicketService.UpdateTicket: %w", err)
	}

	updatedTicket, err := s.ticketRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("TicketService.UpdateTicket: %w", err)
	}

	s.invalidateDashboardCache(ctx)
	return updatedTicket, nil
}

func (s *ticketService) ChangeStatus(ctx context.Context, id uuid.UUID, statusCode string, actorID uuid.UUID, role string) error {
	ticket, err := s.ticketRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("TicketService.ChangeStatus: %w", err)
	}
	if err := ensureTicketAccess(ticket, actorID, role); err != nil {
		return fmt.Errorf("TicketService.ChangeStatus: %w", err)
	}

	if statusCode == "resolved" {
		result := s.db.WithContext(ctx).Exec("CALL sp_resolve_ticket($1, $2)", id, actorID)
		if result.Error != nil {
			return fmt.Errorf("TicketService.ChangeStatus: %w", mapResolveTicketProcedureError(result.Error))
		}
		s.maybeEvaluateResolvedTicket(ctx, id, actorID)
		s.invalidateDashboardCache(ctx)
		return nil
	}

	if err := s.ticketRepo.UpdateStatus(ctx, id, statusCode); err != nil {
		return fmt.Errorf("TicketService.ChangeStatus: %w", err)
	}
	if statusCode == "closed" {
		s.maybeEvaluateResolvedTicket(ctx, id, actorID)
	}
	s.invalidateDashboardCache(ctx)
	return nil
}

func (s *ticketService) AssignTicket(ctx context.Context, ticketID uuid.UUID, agentID uuid.UUID, actorID uuid.UUID) error {
	if ticket, err := s.ticketRepo.GetByID(ctx, ticketID); err == nil {
		if skillErr := s.ensureTicketRequiredSkills(ctx, ticket, actorID); skillErr != nil {
			log.Printf("TicketService.AssignTicket required skills backfill skipped: %v", skillErr)
		}
	}

	justification := "Assigned manually."
	if s.db != nil {
		var agent domain.Agent
		if err := s.db.WithContext(ctx).Preload("User").Where("id = ?", agentID).First(&agent).Error; err == nil {
			activeCounts, countErr := fetchActiveTicketCounts(ctx, s.db, []uuid.UUID{agentID})
			if countErr == nil {
				justification = buildManualAssignmentJustification(&agent, activeCounts[agentID])
			}
		}
	}

	if err := s.callAssignTicketProcedure(ctx, ticketID, agentID, actorID, justification); err != nil {
		return fmt.Errorf("TicketService.AssignTicket: %w", err)
	}
	s.invalidateDashboardCache(ctx)

	return nil
}

func (s *ticketService) invalidateDashboardCache(ctx context.Context) {
	if s.redisCache == nil {
		return
	}
	if err := s.redisCache.Delete(ctx, cache.DashboardStatsCacheKey); err != nil {
		log.Printf("TicketService.invalidateDashboardCache: %v", err)
	}
}

func buildAITriageFailurePayload(err error) []byte {
	reason := "ai triage request failed"
	if err != nil {
		lowerErr := strings.ToLower(err.Error())
		switch {
		case strings.Contains(lowerErr, "unexpected status 404"), strings.Contains(lowerErr, "not_found"):
			reason = "configured ai model is unavailable"
		case strings.Contains(lowerErr, "unexpected status 429"), strings.Contains(lowerErr, "resource_exhausted"), strings.Contains(lowerErr, "quota"):
			reason = "ai provider quota exceeded"
		case strings.Contains(lowerErr, "unexpected status 401"), strings.Contains(lowerErr, "unexpected status 403"), strings.Contains(lowerErr, "permission_denied"), strings.Contains(lowerErr, "api key"):
			reason = "ai provider credentials were rejected"
		case strings.Contains(lowerErr, "context deadline exceeded"), strings.Contains(lowerErr, "client.timeout exceeded"):
			reason = "ai triage request timed out"
		case strings.Contains(lowerErr, "invalid triage json"), strings.Contains(lowerErr, "invalid json response"):
			reason = "ai provider returned invalid json"
		}
	}

	payload, marshalErr := json.Marshal(map[string]string{
		"triage_status": "failed",
		"reason":        reason,
		"details":       err.Error(),
	})
	if marshalErr != nil {
		return nil
	}

	return payload
}

func (s *ticketService) tryAutoAssignTicket(ctx context.Context, ticket *domain.Ticket, actorID uuid.UUID) error {
	if s.db == nil || ticket == nil {
		return nil
	}
	if ticket.AssignedTo != nil {
		return nil
	}

	if err := s.createSkillAuditLog(ctx, ticket.ID, actorID, ticket.RequiredSkills); err != nil {
		return err
	}

	var agents []domain.Agent
	if err := s.db.WithContext(ctx).
		Preload("User").
		Where("is_available = ?", true).
		Find(&agents).Error; err != nil {
		return fmt.Errorf("TicketService.tryAutoAssignTicket: %w", err)
	}

	agentIDs := make([]uuid.UUID, 0, len(agents))
	for _, agent := range agents {
		agentIDs = append(agentIDs, agent.ID)
	}

	activeCounts, err := fetchActiveTicketCounts(ctx, s.db, agentIDs)
	if err != nil {
		return fmt.Errorf("TicketService.tryAutoAssignTicket: %w", err)
	}

	selectedAgent, justification := pickBestAgentForTicket(agents, activeCounts, ticket.RequiredSkills)
	if selectedAgent == nil {
		return nil
	}

	if err := s.callAssignTicketProcedure(ctx, ticket.ID, selectedAgent.ID, actorID, justification); err != nil {
		return fmt.Errorf("TicketService.tryAutoAssignTicket: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"agent_id":                 selectedAgent.ID,
		"agent_name":               selectedAgent.User.FullName,
		"required_skills":          ticket.RequiredSkills,
		"assignment_justification": justification,
		"assignment_strategy":      "skill_match_capacity",
	})
	if err != nil {
		return fmt.Errorf("TicketService.tryAutoAssignTicket: %w", err)
	}

	auditLog := &domain.AuditLog{
		ID:        uuid.New(),
		TicketID:  ticket.ID,
		ActorID:   actorID,
		Action:    "auto_assigned",
		NewValue:  domain.JSONPayload(payload),
		CreatedAt: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(auditLog).Error; err != nil {
		return fmt.Errorf("TicketService.tryAutoAssignTicket: %w", err)
	}

	return nil
}

func (s *ticketService) callAssignTicketProcedure(ctx context.Context, ticketID uuid.UUID, agentID uuid.UUID, actorID uuid.UUID, justification string) error {
	if s.db == nil {
		return fmt.Errorf("assign ticket requires database connection")
	}

	result := s.db.WithContext(ctx).Exec("CALL sp_assign_ticket($1, $2, $3, $4)", ticketID, agentID, actorID, justification)
	if result.Error == nil {
		return nil
	}
	if !shouldFallbackToLegacyAssignProcedure(result.Error) {
		return mapAssignTicketProcedureError(result.Error)
	}

	legacyResult := s.db.WithContext(ctx).Exec("CALL sp_assign_ticket($1, $2, $3)", ticketID, agentID, actorID)
	if legacyResult.Error != nil {
		return mapAssignTicketProcedureError(legacyResult.Error)
	}

	if err := s.persistAssignmentJustificationIfSupported(ctx, ticketID, justification); err != nil {
		log.Printf("TicketService.callAssignTicketProcedure assignment justification persistence skipped: %v", err)
	}
	log.Printf("TicketService.callAssignTicketProcedure using legacy 3-argument sp_assign_ticket; apply migration 005_skill_assignment.sql to enable the newer procedure signature")
	return nil
}

func (s *ticketService) ensureTicketRequiredSkills(ctx context.Context, ticket *domain.Ticket, actorID uuid.UUID) error {
	if ticket == nil || len(ticket.RequiredSkills) > 0 {
		return nil
	}

	requiredSkills := deriveRequiredSkillsForTicket(ticket)
	if len(requiredSkills) == 0 {
		return nil
	}

	ticket.RequiredSkills = requiredSkills
	if err := s.persistRequiredSkillsIfSupported(ctx, ticket.ID, requiredSkills); err != nil {
		return err
	}
	if err := s.createSkillAuditLog(ctx, ticket.ID, actorID, requiredSkills); err != nil {
		return err
	}

	return nil
}

func (s *ticketService) hydrateDerivedTicketMetadata(ticket *domain.Ticket) {
	if ticket == nil {
		return
	}
	if len(ticket.RequiredSkills) == 0 {
		ticket.RequiredSkills = deriveRequiredSkillsForTicket(ticket)
	}
}

func deriveRequiredSkillsForTicket(ticket *domain.Ticket) domain.SkillNameList {
	if ticket == nil {
		return nil
	}

	description := ""
	if ticket.Description != nil {
		description = *ticket.Description
	}

	return resolveRequiredSkills(domain.CreateTicketReq{
		Title:          ticket.Title,
		Description:    description,
		RequiredSkills: []string(ticket.RequiredSkills),
	}, json.RawMessage(ticket.AIInsights))
}

func (s *ticketService) persistAssignmentJustificationIfSupported(ctx context.Context, ticketID uuid.UUID, justification string) error {
	if s.db == nil {
		return nil
	}

	trimmed := strings.TrimSpace(justification)
	if trimmed == "" {
		return nil
	}

	result := s.db.WithContext(ctx).
		Model(&domain.Ticket{}).
		Where("id = ?", ticketID).
		Update("assignment_justification", trimmed)
	if result.Error == nil || isMissingAssignmentJustificationColumn(result.Error) {
		return nil
	}

	return result.Error
}

func (s *ticketService) persistRequiredSkillsIfSupported(ctx context.Context, ticketID uuid.UUID, requiredSkills domain.SkillNameList) error {
	if s.db == nil || len(requiredSkills) == 0 {
		return nil
	}

	result := s.db.WithContext(ctx).
		Model(&domain.Ticket{}).
		Where("id = ?", ticketID).
		Update("required_skills", requiredSkills)
	if result.Error == nil || isMissingRequiredSkillsColumn(result.Error) {
		return nil
	}

	return result.Error
}

func (s *ticketService) createSkillAuditLog(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID, requiredSkills domain.SkillNameList) error {
	if s.db == nil || len(requiredSkills) == 0 {
		return nil
	}

	payload, err := json.Marshal(map[string]any{
		"required_skills": requiredSkills,
	})
	if err != nil {
		return err
	}

	auditLog := &domain.AuditLog{
		ID:        uuid.New(),
		TicketID:  ticketID,
		ActorID:   actorID,
		Action:    "required_skills_identified",
		NewValue:  domain.JSONPayload(payload),
		CreatedAt: time.Now(),
	}
	return s.db.WithContext(ctx).Create(auditLog).Error
}

func (s *ticketService) maybeEvaluateResolvedTicket(ctx context.Context, ticketID uuid.UUID, actorID uuid.UUID) {
	if s.db == nil {
		return
	}

	ticket, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil || ticket == nil {
		if err != nil {
			log.Printf("TicketService.maybeEvaluateResolvedTicket load failed: %v", err)
		}
		return
	}
	if ticket.Assignee == nil || ticket.SkillsEvaluatedAt != nil || len(ticket.RequiredSkills) == 0 {
		return
	}

	updatedSkills := deriveUpdatedSkillScores(ticket.Assignee.Skills, ticket.RequiredSkills, ticket.Priority.Name, ticket.SLABreached)

	oldPayload, err := json.Marshal(ticket.Assignee.Skills)
	if err != nil {
		log.Printf("TicketService.maybeEvaluateResolvedTicket marshal old skills failed: %v", err)
		return
	}
	newPayload, err := json.Marshal(map[string]any{
		"agent_id":        ticket.Assignee.ID,
		"agent_name":      ticket.Assignee.User.FullName,
		"required_skills": ticket.RequiredSkills,
		"skills":          updatedSkills,
		"sla_breached":    ticket.SLABreached,
	})
	if err != nil {
		log.Printf("TicketService.maybeEvaluateResolvedTicket marshal new skills failed: %v", err)
		return
	}

	now := time.Now()
	if txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.Agent{}).Where("id = ?", ticket.Assignee.ID).Update("skills", updatedSkills).Error; err != nil {
			return fmt.Errorf("update agent skills: %w", err)
		}

		if err := tx.Model(&domain.Ticket{}).Where("id = ?", ticket.ID).Update("skills_evaluated_at", now).Error; err != nil {
			return fmt.Errorf("mark ticket skills evaluated: %w", err)
		}

		auditLog := &domain.AuditLog{
			ID:        uuid.New(),
			TicketID:  ticket.ID,
			ActorID:   actorID,
			Action:    "skills_evaluated",
			OldValue:  domain.JSONPayload(oldPayload),
			NewValue:  domain.JSONPayload(newPayload),
			CreatedAt: now,
		}
		if err := tx.Create(auditLog).Error; err != nil {
			return fmt.Errorf("insert skill evaluation audit: %w", err)
		}
		return nil
	}); txErr != nil {
		log.Printf("TicketService.maybeEvaluateResolvedTicket failed: %v", txErr)
	}
}

func mapAssignTicketProcedureError(err error) error {
	if err == nil {
		return nil
	}

	lowerErr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lowerErr, "agent is not available"):
		return &domain.AppError{Code: domain.ErrBadRequest.Code, Message: "agent is not available", Err: err}
	case strings.Contains(lowerErr, "maximum ticket limit"):
		return &domain.AppError{Code: domain.ErrBadRequest.Code, Message: "agent has reached maximum ticket limit", Err: err}
	case strings.Contains(lowerErr, "cannot assign"):
		return &domain.AppError{Code: domain.ErrBadRequest.Code, Message: "ticket cannot be assigned in its current status", Err: err}
	default:
		return err
	}
}

func shouldFallbackToLegacyAssignProcedure(err error) bool {
	if err == nil {
		return false
	}

	lowerErr := strings.ToLower(err.Error())
	return strings.Contains(lowerErr, "procedure sp_assign_ticket") &&
		strings.Contains(lowerErr, "does not exist") &&
		(strings.Contains(lowerErr, "sqlstate 42883") || strings.Contains(lowerErr, "unknown"))
}

func isMissingAssignmentJustificationColumn(err error) bool {
	if err == nil {
		return false
	}

	lowerErr := strings.ToLower(err.Error())
	return strings.Contains(lowerErr, "assignment_justification") &&
		strings.Contains(lowerErr, "does not exist")
}

func isMissingRequiredSkillsColumn(err error) bool {
	if err == nil {
		return false
	}

	lowerErr := strings.ToLower(err.Error())
	return strings.Contains(lowerErr, "required_skills") &&
		strings.Contains(lowerErr, "does not exist")
}

func mapResolveTicketProcedureError(err error) error {
	if err == nil {
		return nil
	}

	lowerErr := strings.ToLower(err.Error())
	if strings.Contains(lowerErr, "cannot resolve") {
		return &domain.AppError{Code: domain.ErrBadRequest.Code, Message: "ticket cannot be resolved in its current status", Err: err}
	}

	return err
}

func isTerminalTicketStatus(status string) bool {
	switch status {
	case "resolved", "closed", "cancelled":
		return true
	default:
		return false
	}
}

func buildClassificationChangeComment(ticket *domain.Ticket, updatedPriorityName, updatedCategoryName, reason string, priorityChanged, categoryChanged bool) string {
	lines := []string{"Ticket classification updated."}

	if priorityChanged {
		lines = append(lines, fmt.Sprintf("Priority: %s -> %s", ticket.Priority.Name, updatedPriorityName))
	}
	if categoryChanged {
		lines = append(lines, fmt.Sprintf("Category: %s -> %s", ticket.Category.Name, updatedCategoryName))
	}

	lines = append(lines, fmt.Sprintf("Reason: %s", reason))
	return strings.Join(lines, "\n")
}
