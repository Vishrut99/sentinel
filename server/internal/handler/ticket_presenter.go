package handler

import "github.com/yourusername/incident-ticketing/internal/domain"

func toTicketResponse(ticket *domain.Ticket) domain.TicketRes {
	description := ""
	if ticket.Description != nil {
		description = *ticket.Description
	}

	response := domain.TicketRes{
		ID:             ticket.ID,
		TicketNumber:   ticket.TicketNumber,
		Title:          ticket.Title,
		Description:    description,
		AIInsights:     ticket.AIInsights,
		RequiredSkills: []string(ticket.RequiredSkills),
		Status:         ticket.Status.Name,
		Priority:       ticket.Priority.Name,
		CategoryID:     ticket.CategoryID,
		Category:       ticket.Category.Name,
		CreatedBy: domain.UserSummary{
			ID:       ticket.Creator.ID,
			Email:    ticket.Creator.Email,
			FullName: ticket.Creator.FullName,
			Role:     ticket.Creator.Role,
		},
		AssignmentJustification: ticket.AssignmentJustification,
		IsProblem:               ticket.IsProblem,
		ParentID:                ticket.ParentID,
		SLABreached:             ticket.SLABreached,
		DueAt:                   ticket.DueAt,
		ResolvedAt:              ticket.ResolvedAt,
		CreatedAt:               ticket.CreatedAt,
		UpdatedAt:               ticket.UpdatedAt,
	}

	if ticket.Assignee != nil {
		department := ""
		if ticket.Assignee.Department != nil {
			department = *ticket.Assignee.Department
		}
		response.AssignedAgent = &domain.AgentSummary{
			ID:                 ticket.Assignee.ID,
			FullName:           ticket.Assignee.User.FullName,
			Department:         department,
			IsAvailable:        ticket.Assignee.IsAvailable,
			MaxTickets:         ticket.Assignee.MaxTickets,
			Skills:             ticket.Assignee.Skills,
			ActiveTickets:      ticket.Assignee.ActiveTickets,
			UtilizationPercent: 0,
		}
	}

	return response
}

func toTicketListResponse(tickets []domain.Ticket, total int64, page, perPage int) domain.TicketListRes {
	response := domain.TicketListRes{
		Data:    make([]domain.TicketRes, 0, len(tickets)),
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}

	for i := range tickets {
		ticket := tickets[i]
		response.Data = append(response.Data, toTicketResponse(&ticket))
	}

	return response
}
