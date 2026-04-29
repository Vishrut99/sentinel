package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/yourusername/incident-ticketing/internal/domain"
)

type TicketRepository interface {
	Create(ctx context.Context, ticket *domain.Ticket) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Ticket, error)
	List(ctx context.Context, filter domain.TicketListFilter) ([]domain.Ticket, int64, error)
	SetParent(ctx context.Context, ticketID uuid.UUID, parentID uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, statusCode string) error
	GetPriorityByName(ctx context.Context, name string) (*domain.Priority, error)
	GetStatusByName(ctx context.Context, name string) (*domain.TicketStatus, error)
	MarkSLABreached(ctx context.Context) error
}

type ticketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) TicketRepository {
	return &ticketRepository{db: db}
}

func (r *ticketRepository) GetPriorityByName(ctx context.Context, name string) (*domain.Priority, error) {
	var priority domain.Priority
	result := r.db.WithContext(ctx).Where("name = ?", name).First(&priority)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("TicketRepository.GetPriorityByName: %w", result.Error)
	}
	return &priority, nil
}

func (r *ticketRepository) GetStatusByName(ctx context.Context, name string) (*domain.TicketStatus, error) {
	var status domain.TicketStatus
	result := r.db.WithContext(ctx).Where("name = ?", name).First(&status)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("TicketRepository.GetStatusByName: %w", result.Error)
	}
	return &status, nil
}

func (r *ticketRepository) Create(ctx context.Context, ticket *domain.Ticket) error {
	if err := r.db.WithContext(ctx).Create(ticket).Error; err != nil {
		return fmt.Errorf("TicketRepository.Create: %w", err)
	}
	return nil
}

// we do preload so when we get id we get simple fields like id,ticket id etc but if we want to see realted objects we use it
//SELECT * FROM ticket_statuses WHERE id = 2 like this for .Preload("Status")

func (r *ticketRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Ticket, error) {
	var ticket domain.Ticket
	result := r.db.WithContext(ctx).Where("id = ?", id).
		Preload("Status").
		Preload("Priority").
		Preload("Category").
		Preload("Creator").
		Preload("Assignee").
		Preload("Assignee.User").
		First(&ticket)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound // ticket doesn't exist
		}
		return nil, fmt.Errorf("TicketRepository.GetByID: %w", result.Error)
	}
	return &ticket, nil
}

func (r *ticketRepository) List(ctx context.Context, filter domain.TicketListFilter) ([]domain.Ticket, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Ticket{}) // we are using ticket table without fetchin any data yet
	joinedStatuses := false
	joinedPriorities := false
	joinedCategories := false
	joinedAssigneeUsers := false

	if filter.Status != "" {
		query = query.Joins("Join ticket_statuses ON ticket_statuses.id =tickets.status_id").Where("ticket_statuses.name = ?", filter.Status)
		joinedStatuses = true
	}
	if filter.Priority != "" {
		query = query.Joins("JOIN priorities ON priorities.id = tickets.priority_id").
			Where("priorities.name = ?", filter.Priority)
		joinedPriorities = true
	}
	if filter.CategoryID != uuid.Nil {
		query = query.Where("category_id =?", filter.CategoryID)
	}
	if trimmedQuery := strings.TrimSpace(filter.Query); trimmedQuery != "" {
		searchTerm := "%" + trimmedQuery + "%"
		query = query.Where(
			`tickets.title ILIKE ?
				OR tickets.ticket_number ILIKE ?
				OR COALESCE(tickets.description, '') ILIKE ?
				OR COALESCE(tickets.assignment_justification, '') ILIKE ?
				OR COALESCE(CAST(tickets.required_skills AS text), '') ILIKE ?
				OR COALESCE(CAST(tickets.ai_insights AS text), '') ILIKE ?`,
			searchTerm,
			searchTerm,
			searchTerm,
			searchTerm,
			searchTerm,
			searchTerm,
		)
	}
	if filter.IsProblem != nil {
		query = query.Where("tickets.is_problem = ?", *filter.IsProblem)
	}
	if filter.SLABreached != nil {
		query = query.Where("sla_breached=?", *filter.SLABreached)
	}
	if filter.CreatedBy != uuid.Nil {
		query = query.Where("created_by = ?", filter.CreatedBy)
	}
	if filter.AssignedUserID != uuid.Nil {
		query = query.Joins("JOIN agents assigned_agents ON assigned_agents.id = tickets.assigned_to").
			Where("assigned_agents.user_id = ?", filter.AssignedUserID)
	}
	if filter.AssignedAgentID != uuid.Nil {
		query = query.Where("tickets.assigned_to = ?", filter.AssignedAgentID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("TicketRepository.List: %w", err)
	}

	sortDirection := "desc"
	if strings.EqualFold(filter.SortDirection, "asc") {
		sortDirection = "asc"
	}

	switch filter.SortBy {
	case "ticket_number":
		query = query.Order("tickets.ticket_number " + sortDirection)
	case "title":
		query = query.Order("tickets.title " + sortDirection)
	case "created_at":
		query = query.Order("tickets.created_at " + sortDirection)
	case "updated_at":
		query = query.Order("tickets.updated_at " + sortDirection)
	case "due_at":
		query = query.Order("tickets.due_at " + sortDirection)
	case "category":
		if !joinedCategories {
			query = query.Joins("JOIN categories ON categories.id = tickets.category_id")
			joinedCategories = true
		}
		query = query.Order("categories.name " + sortDirection)
	case "priority":
		if !joinedPriorities {
			query = query.Joins("JOIN priorities ON priorities.id = tickets.priority_id")
		}
		query = query.Order(`
			CASE priorities.name
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				WHEN 'low' THEN 4
				ELSE 5
			END ` + sortDirection)
	case "status":
		if !joinedStatuses {
			query = query.Joins("JOIN ticket_statuses ON ticket_statuses.id = tickets.status_id")
		}
		query = query.Order("ticket_statuses.name " + sortDirection)
	case "assigned_to":
		if !joinedAssigneeUsers {
			query = query.
				Joins("LEFT JOIN agents sort_agents ON sort_agents.id = tickets.assigned_to").
				Joins("LEFT JOIN users sort_assignee_users ON sort_assignee_users.id = sort_agents.user_id")
			joinedAssigneeUsers = true
		}
		query = query.Order("sort_assignee_users.full_name " + sortDirection)
	case "sla_breached":
		query = query.Order("tickets.sla_breached " + sortDirection)
	default:
		query = query.Order("tickets.created_at desc")
	}

	query = query.Offset((filter.Page - 1) * filter.PerPage).Limit(filter.PerPage)

	var tickets []domain.Ticket
	if err := query.Preload("Status").Preload("Priority").Preload("Category").Preload("Creator").Preload("Assignee").Preload("Assignee.User").
		Find(&tickets).Error; err != nil {
		return nil, 0, fmt.Errorf("TicketRepository.List: %w", err)
	}
	return tickets, total, nil
}

func (r *ticketRepository) UpdateStatus(ctx context.Context, id uuid.UUID, statusCode string) error {
	var status domain.TicketStatus
	if err := r.db.WithContext(ctx).Where("name = ?", statusCode).First(&status).Error; err != nil {
		return fmt.Errorf("TicketRepository.UpdateStatus: %w", err)
	}
	if err := r.db.WithContext(ctx).Model(&domain.Ticket{}).
		Where("id = ?", id).
		Update("status_id", status.ID).Error; err != nil {
		return fmt.Errorf("TicketRepository.UpdateStatus: %w", err)
	}
	return nil
}

func (r *ticketRepository) SetParent(ctx context.Context, ticketID uuid.UUID, parentID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Model(&domain.Ticket{}).
		Where("id = ?", ticketID).
		Update("parent_id", parentID).Error; err != nil {
		return fmt.Errorf("TicketRepository.SetParent: %w", err)
	}
	return nil
}

func (r *ticketRepository) MarkSLABreached(ctx context.Context) error {
	return r.db.WithContext(ctx).Model(&domain.Ticket{}).
		Where("due_at < ?", time.Now()).
		Where("sla_breached = ?", false).
		Where("status_id NOT IN (?)",
			r.db.Model(&domain.TicketStatus{}).
				Select("id").
				Where("name IN ?", []string{"resolved", "closed", "cancelled"}),
		).
		Update("sla_breached", true).Error
}
