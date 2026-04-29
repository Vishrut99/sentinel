package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
	"gorm.io/gorm"
)

type DashboardRepository interface {
	GetDashboardStats(ctx context.Context, filter domain.TicketListFilter) (*domain.DashboardStatsRes, error)
}

type dashboardRepository struct {
	db *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) DashboardRepository {
	return &dashboardRepository{db: db}
}

func (r *dashboardRepository) filteredQuery(ctx context.Context, filter domain.TicketListFilter) *gorm.DB {
	query := r.db.WithContext(ctx).Model(&domain.Ticket{})

	if filter.Status != "" {
		query = query.Where(
			"tickets.status_id IN (?)",
			r.db.WithContext(ctx).Model(&domain.TicketStatus{}).Select("id").Where("name = ?", filter.Status),
		)
	}
	if filter.Priority != "" {
		query = query.Where(
			"tickets.priority_id IN (?)",
			r.db.WithContext(ctx).Model(&domain.Priority{}).Select("id").Where("name = ?", filter.Priority),
		)
	}
	if filter.CategoryID != uuid.Nil {
		query = query.Where("tickets.category_id = ?", filter.CategoryID)
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
		query = query.Where("tickets.sla_breached = ?", *filter.SLABreached)
	}
	if filter.CreatedBy != uuid.Nil {
		query = query.Where("tickets.created_by = ?", filter.CreatedBy)
	}
	if filter.AssignedUserID != uuid.Nil {
		query = query.Joins("JOIN agents assigned_agents ON assigned_agents.id = tickets.assigned_to").
			Where("assigned_agents.user_id = ?", filter.AssignedUserID)
	}
	if filter.AssignedAgentID != uuid.Nil {
		query = query.Where("tickets.assigned_to = ?", filter.AssignedAgentID)
	}

	return query
}

func (r *dashboardRepository) GetDashboardStats(ctx context.Context, filter domain.TicketListFilter) (*domain.DashboardStatsRes, error) {
	var totalTickets int64
	var openTickets int64
	var inProgressTickets int64
	var resolvedTickets int64
	var slaBreached int64
	var avgResolveHours float64

	if err := r.filteredQuery(ctx, filter).Count(&totalTickets).Error; err != nil {
		return nil, fmt.Errorf("dashboardRepository.GetDashboardStats counts: %w", err)
	}

	if err := r.filteredQuery(ctx, filter).
		Joins("JOIN ticket_statuses ts ON ts.id = tickets.status_id").
		Where("ts.name = ?", "open").Count(&openTickets).Error; err != nil {
		return nil, fmt.Errorf("dashboardRepository.GetDashboardStats open: %w", err)
	}

	if err := r.filteredQuery(ctx, filter).
		Joins("JOIN ticket_statuses ts ON ts.id = tickets.status_id").
		Where("ts.name = ?", "in_progress").Count(&inProgressTickets).Error; err != nil {
		return nil, fmt.Errorf("dashboardRepository.GetDashboardStats in_progress: %w", err)
	}

	if err := r.filteredQuery(ctx, filter).
		Joins("JOIN ticket_statuses ts ON ts.id = tickets.status_id").
		Where("ts.name = ?", "resolved").Count(&resolvedTickets).Error; err != nil {
		return nil, fmt.Errorf("dashboardRepository.GetDashboardStats resolved: %w", err)
	}

	if err := r.filteredQuery(ctx, filter).Where("tickets.sla_breached = ?", true).Count(&slaBreached).Error; err != nil {
		return nil, fmt.Errorf("dashboardRepository.GetDashboardStats sla: %w", err)
	}

	if err := r.filteredQuery(ctx, filter).
		Where("tickets.resolved_at IS NOT NULL").
		Select("COALESCE(AVG(EXTRACT(EPOCH FROM (tickets.resolved_at - tickets.created_at)) / 3600), 0)").
		Scan(&avgResolveHours).Error; err != nil {
		return nil, fmt.Errorf("dashboardRepository.GetDashboardStats avg_resolve: %w", err)
	}

	var priorityCounts []struct {
		Name  string
		Count int64
	}
	if err := r.filteredQuery(ctx, filter).
		Select("priorities.name, count(*) as count").
		Joins("JOIN priorities ON priorities.id = tickets.priority_id").
		Group("priorities.name").
		Scan(&priorityCounts).Error; err != nil {
		return nil, fmt.Errorf("dashboardRepository.GetDashboardStats priority: %w", err)
	}

	byPriority := make(map[string]int64)
	for _, pc := range priorityCounts {
		byPriority[pc.Name] = pc.Count
	}

	var categoryCounts []struct {
		Name  string
		Count int64
	}
	if err := r.filteredQuery(ctx, filter).
		Select("categories.name, count(*) as count").
		Joins("JOIN categories ON categories.id = tickets.category_id").
		Group("categories.name").
		Scan(&categoryCounts).Error; err != nil {
		return nil, fmt.Errorf("dashboardRepository.GetDashboardStats category: %w", err)
	}

	byCategory := make(map[string]int64)
	for _, cc := range categoryCounts {
		byCategory[cc.Name] = cc.Count
	}

	return &domain.DashboardStatsRes{
		TotalTickets:      totalTickets,
		OpenTickets:       openTickets,
		InProgressTickets: inProgressTickets,
		ResolvedToday:     resolvedTickets, // storing 'resolved' in resolvedToday per struct
		SLABreached:       slaBreached,
		AvgResolveHours:   avgResolveHours,
		ByPriority:        byPriority,
		ByCategory:        byCategory,
	}, nil
}
