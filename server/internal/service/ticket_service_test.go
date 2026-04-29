package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
)

type fakeTicketRepository struct {
	createdTicket  *domain.Ticket
	ticketsByID    map[uuid.UUID]*domain.Ticket
	lastListFilter domain.TicketListFilter
	statusUpdates  []struct {
		id     uuid.UUID
		status string
	}
	setParentArgs struct {
		ticketID uuid.UUID
		parentID uuid.UUID
		called   bool
	}
}

func (f *fakeTicketRepository) Create(ctx context.Context, ticket *domain.Ticket) error {
	f.createdTicket = ticket
	if f.ticketsByID == nil {
		f.ticketsByID = make(map[uuid.UUID]*domain.Ticket)
	}
	f.ticketsByID[ticket.ID] = ticket
	return nil
}

func (f *fakeTicketRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Ticket, error) {
	if ticket, ok := f.ticketsByID[id]; ok {
		return ticket, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeTicketRepository) List(ctx context.Context, filter domain.TicketListFilter) ([]domain.Ticket, int64, error) {
	f.lastListFilter = filter
	if len(f.ticketsByID) == 0 {
		return nil, 0, nil
	}

	tickets := make([]domain.Ticket, 0, len(f.ticketsByID))
	for _, ticket := range f.ticketsByID {
		tickets = append(tickets, *ticket)
	}
	return tickets, int64(len(tickets)), nil
}

func (f *fakeTicketRepository) SetParent(ctx context.Context, ticketID uuid.UUID, parentID uuid.UUID) error {
	f.setParentArgs.ticketID = ticketID
	f.setParentArgs.parentID = parentID
	f.setParentArgs.called = true
	return nil
}

func (f *fakeTicketRepository) UpdateStatus(ctx context.Context, id uuid.UUID, statusCode string) error {
	f.statusUpdates = append(f.statusUpdates, struct {
		id     uuid.UUID
		status string
	}{id: id, status: statusCode})
	return nil
}

func (f *fakeTicketRepository) GetPriorityByName(ctx context.Context, name string) (*domain.Priority, error) {
	return &domain.Priority{ID: 1, Name: name}, nil
}

func (f *fakeTicketRepository) GetStatusByName(ctx context.Context, name string) (*domain.TicketStatus, error) {
	return &domain.TicketStatus{ID: 1, Name: name}, nil
}

func (f *fakeTicketRepository) MarkSLABreached(ctx context.Context) error {
	return nil
}

type fakeAITriageService struct {
	result json.RawMessage
	err    error
}

func (f *fakeAITriageService) AnalyzeTicket(ctx context.Context, title, description string) (json.RawMessage, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func TestTicketServiceCreateTicketSavesAIInsights(t *testing.T) {
	repo := &fakeTicketRepository{ticketsByID: make(map[uuid.UUID]*domain.Ticket)}
	ai := &fakeAITriageService{result: json.RawMessage(`{"suggested_priority":"high"}`)}
	service := NewTicketService(repo, nil, nil, ai)

	request := domain.CreateTicketReq{
		Title:       "VPN timeout on login",
		Description: "Unable to connect to VPN for 30 minutes",
		Priority:    "high",
		CategoryID:  uuid.New(),
	}

	_, err := service.CreateTicket(context.Background(), request, uuid.New(), "user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.createdTicket == nil {
		t.Fatalf("expected ticket to be created")
	}
	if string(repo.createdTicket.AIInsights) != `{"suggested_priority":"high"}` {
		t.Fatalf("expected ai insights to be stored, got %s", string(repo.createdTicket.AIInsights))
	}
}

func TestTicketServiceCreateTicketUserCannotCreateProblem(t *testing.T) {
	repo := &fakeTicketRepository{ticketsByID: make(map[uuid.UUID]*domain.Ticket)}
	service := NewTicketService(repo, nil, nil, nil)

	request := domain.CreateTicketReq{
		Title:      "Major outage",
		Priority:   "critical",
		CategoryID: uuid.New(),
		IsProblem:  true,
	}

	_, err := service.CreateTicket(context.Background(), request, uuid.New(), "user")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestTicketServiceCreateTicketStoresReadableAITriageFailureReason(t *testing.T) {
	repo := &fakeTicketRepository{ticketsByID: make(map[uuid.UUID]*domain.Ticket)}
	ai := &fakeAITriageService{err: errors.New("AITriageService.callGeminiEndpoint: unexpected status 404 (NOT_FOUND): model is not supported")}
	service := NewTicketService(repo, nil, nil, ai)

	request := domain.CreateTicketReq{
		Title:       "VPN timeout on login",
		Description: "Unable to connect to VPN for 30 minutes",
		Priority:    "high",
		CategoryID:  uuid.New(),
	}

	_, err := service.CreateTicket(context.Background(), request, uuid.New(), "user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(repo.createdTicket.AIInsights, &payload); err != nil {
		t.Fatalf("expected valid JSON payload, got %v", err)
	}
	if payload["reason"] != "configured ai model is unavailable" {
		t.Fatalf("expected readable failure reason, got %q", payload["reason"])
	}
	if payload["triage_status"] != "failed" {
		t.Fatalf("expected failed triage status, got %q", payload["triage_status"])
	}
}

func TestTicketServiceLinkToProblemParentMustBeProblem(t *testing.T) {
	repo := &fakeTicketRepository{ticketsByID: make(map[uuid.UUID]*domain.Ticket)}
	service := NewTicketService(repo, nil, nil, nil)

	ticketID := uuid.New()
	parentID := uuid.New()
	repo.ticketsByID[ticketID] = &domain.Ticket{ID: ticketID, IsProblem: false}
	repo.ticketsByID[parentID] = &domain.Ticket{ID: parentID, IsProblem: false}

	err := service.LinkToProblem(context.Background(), ticketID, domain.LinkProblemReq{ParentID: parentID}, uuid.New(), "admin")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrBadRequest) {
		t.Fatalf("expected bad request error, got %v", err)
	}
}

func TestTicketServiceLinkToProblemSuccess(t *testing.T) {
	repo := &fakeTicketRepository{ticketsByID: make(map[uuid.UUID]*domain.Ticket)}
	service := NewTicketService(repo, nil, nil, nil)

	ticketID := uuid.New()
	parentID := uuid.New()
	repo.ticketsByID[ticketID] = &domain.Ticket{ID: ticketID, IsProblem: false}
	repo.ticketsByID[parentID] = &domain.Ticket{ID: parentID, IsProblem: true}

	err := service.LinkToProblem(context.Background(), ticketID, domain.LinkProblemReq{ParentID: parentID}, uuid.New(), "admin")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.setParentArgs.called {
		t.Fatalf("expected SetParent to be called")
	}
	if repo.setParentArgs.ticketID != ticketID || repo.setParentArgs.parentID != parentID {
		t.Fatalf("unexpected set parent args: ticket=%s parent=%s", repo.setParentArgs.ticketID, repo.setParentArgs.parentID)
	}
}

func TestTicketServiceGetTicketForbidsAgentWhenTicketIsNotAssigned(t *testing.T) {
	agentUserID := uuid.New()
	ticketID := uuid.New()
	repo := &fakeTicketRepository{
		ticketsByID: map[uuid.UUID]*domain.Ticket{
			ticketID: {ID: ticketID, CreatedBy: uuid.New()},
		},
	}
	service := NewTicketService(repo, nil, nil, nil)

	_, err := service.GetTicket(context.Background(), ticketID, agentUserID, "agent")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestTicketServiceGetTicketAllowsAssignedAgent(t *testing.T) {
	agentUserID := uuid.New()
	ticketID := uuid.New()
	repo := &fakeTicketRepository{
		ticketsByID: map[uuid.UUID]*domain.Ticket{
			ticketID: {ID: ticketID, CreatedBy: uuid.New(), Assignee: &domain.Agent{UserID: agentUserID}},
		},
	}
	service := NewTicketService(repo, nil, nil, nil)

	ticket, err := service.GetTicket(context.Background(), ticketID, agentUserID, "agent")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ticket == nil || ticket.ID != ticketID {
		t.Fatalf("expected assigned ticket, got %#v", ticket)
	}
}

func TestTicketServiceGetTicketHydratesRequiredSkillsForLegacyTicket(t *testing.T) {
	userID := uuid.New()
	ticketID := uuid.New()
	repo := &fakeTicketRepository{
		ticketsByID: map[uuid.UUID]*domain.Ticket{
			ticketID: {
				ID:        ticketID,
				CreatedBy: userID,
				Title:     "VPN access fails after MFA",
				Description: func() *string {
					value := "Remote users cannot complete VPN sign in after the MFA step."
					return &value
				}(),
			},
		},
	}
	service := NewTicketService(repo, nil, nil, nil)

	ticket, err := service.GetTicket(context.Background(), ticketID, userID, "user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(ticket.RequiredSkills) == 0 {
		t.Fatalf("expected required skills to be hydrated for legacy ticket")
	}
}

func TestTicketServiceListTicketFiltersAgentsToAssignedTickets(t *testing.T) {
	agentUserID := uuid.New()
	repo := &fakeTicketRepository{ticketsByID: make(map[uuid.UUID]*domain.Ticket)}
	service := NewTicketService(repo, nil, nil, nil)

	_, _, err := service.ListTicket(context.Background(), domain.TicketListFilter{Page: 1, PerPage: 10}, agentUserID, "agent")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.lastListFilter.AssignedUserID != agentUserID {
		t.Fatalf("expected assigned user filter to be set, got %s", repo.lastListFilter.AssignedUserID)
	}
	if repo.lastListFilter.CreatedBy != uuid.Nil {
		t.Fatalf("expected created_by to remain empty for agents, got %s", repo.lastListFilter.CreatedBy)
	}
}

func TestTicketServiceListTicketHydratesRequiredSkillsForLegacyTickets(t *testing.T) {
	userID := uuid.New()
	ticketID := uuid.New()
	repo := &fakeTicketRepository{
		ticketsByID: map[uuid.UUID]*domain.Ticket{
			ticketID: {
				ID:        ticketID,
				CreatedBy: userID,
				Title:     "Laptop shuts down during boot",
				Description: func() *string {
					value := "User reports the workstation powers off repeatedly while starting."
					return &value
				}(),
			},
		},
	}
	service := NewTicketService(repo, nil, nil, nil)

	tickets, _, err := service.ListTicket(context.Background(), domain.TicketListFilter{Page: 1, PerPage: 10}, userID, "user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected one ticket, got %d", len(tickets))
	}
	if len(tickets[0].RequiredSkills) == 0 {
		t.Fatalf("expected required skills to be hydrated in list response")
	}
}

func TestTicketServiceChangeStatusForbidsUnassignedAgent(t *testing.T) {
	agentUserID := uuid.New()
	ticketID := uuid.New()
	repo := &fakeTicketRepository{
		ticketsByID: map[uuid.UUID]*domain.Ticket{
			ticketID: {ID: ticketID, CreatedBy: uuid.New()},
		},
	}
	service := NewTicketService(repo, nil, nil, nil)

	err := service.ChangeStatus(context.Background(), ticketID, "in_progress", agentUserID, "agent")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
	if len(repo.statusUpdates) != 0 {
		t.Fatalf("expected no status update, got %+v", repo.statusUpdates)
	}
}

func TestTicketServiceChangeStatusAllowsAssignedAgent(t *testing.T) {
	agentUserID := uuid.New()
	ticketID := uuid.New()
	repo := &fakeTicketRepository{
		ticketsByID: map[uuid.UUID]*domain.Ticket{
			ticketID: {ID: ticketID, CreatedBy: uuid.New(), Assignee: &domain.Agent{UserID: agentUserID}},
		},
	}
	service := NewTicketService(repo, nil, nil, nil)

	err := service.ChangeStatus(context.Background(), ticketID, "in_progress", agentUserID, "agent")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repo.statusUpdates) != 1 {
		t.Fatalf("expected one status update, got %+v", repo.statusUpdates)
	}
	if repo.statusUpdates[0].status != "in_progress" {
		t.Fatalf("expected in_progress update, got %+v", repo.statusUpdates[0])
	}
}

func TestEnsureTicketRequiredSkillsBackfillsLegacyTicketMetadata(t *testing.T) {
	repo := &fakeTicketRepository{ticketsByID: make(map[uuid.UUID]*domain.Ticket)}
	service := NewTicketService(repo, nil, nil, nil)

	ticket := &domain.Ticket{
		ID:    uuid.New(),
		Title: "VPN access fails after MFA",
		Description: func() *string {
			value := "Remote users cannot complete VPN sign in after the MFA step."
			return &value
		}(),
	}

	if err := service.(*ticketService).ensureTicketRequiredSkills(context.Background(), ticket, uuid.New()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(ticket.RequiredSkills) == 0 {
		t.Fatalf("expected required skills to be inferred for legacy ticket")
	}
}

func TestMapAssignTicketProcedureErrorReturnsBadRequestForCapacityLimit(t *testing.T) {
	err := mapAssignTicketProcedureError(errors.New("ERROR: Agent has reached maximum ticket limit"))

	var appErr *domain.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %v", err)
	}
	if appErr.Code != domain.ErrBadRequest.Code {
		t.Fatalf("expected bad request code, got %s", appErr.Code)
	}
	if appErr.Message != "agent has reached maximum ticket limit" {
		t.Fatalf("unexpected message: %s", appErr.Message)
	}
}

func TestShouldFallbackToLegacyAssignProcedureReturnsTrueForLegacyProcedureSignature(t *testing.T) {
	err := errors.New("ERROR: procedure sp_assign_ticket(unknown, unknown, unknown, unknown) does not exist (SQLSTATE 42883)")

	if !shouldFallbackToLegacyAssignProcedure(err) {
		t.Fatalf("expected legacy assign procedure fallback to be enabled")
	}
}

func TestShouldFallbackToLegacyAssignProcedureReturnsFalseForBusinessError(t *testing.T) {
	err := errors.New("ERROR: Agent has reached maximum ticket limit")

	if shouldFallbackToLegacyAssignProcedure(err) {
		t.Fatalf("expected no fallback for normal procedure business error")
	}
}

func TestMapResolveTicketProcedureErrorReturnsBadRequestForInvalidStatus(t *testing.T) {
	err := mapResolveTicketProcedureError(errors.New("ERROR: Cannot resolve a closed ticket"))

	var appErr *domain.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %v", err)
	}
	if appErr.Code != domain.ErrBadRequest.Code {
		t.Fatalf("expected bad request code, got %s", appErr.Code)
	}
	if appErr.Message != "ticket cannot be resolved in its current status" {
		t.Fatalf("unexpected message: %s", appErr.Message)
	}
}
