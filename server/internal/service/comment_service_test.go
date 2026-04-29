package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
)

type fakeCommentRepository struct {
	createdComment      *domain.Comment
	listedComments      []domain.Comment
	listIncludeInternal bool
	listTicketID        uuid.UUID
	createErr           error
	listErr             error
}

func (f *fakeCommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.createdComment = comment
	return nil
}

func (f *fakeCommentRepository) ListByTicketID(ctx context.Context, ticketID uuid.UUID, includeInternal bool) ([]domain.Comment, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	f.listTicketID = ticketID
	f.listIncludeInternal = includeInternal
	return f.listedComments, nil
}

type fakeCommentTicketReader struct {
	ticket *domain.Ticket
	err    error
}

func (f *fakeCommentTicketReader) GetByID(ctx context.Context, id uuid.UUID) (*domain.Ticket, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.ticket, nil
}

func TestCommentServiceUserCanCreateExternalCommentOnOwnTicket(t *testing.T) {
	ownerID := uuid.New()
	ticketID := uuid.New()
	commentRepo := &fakeCommentRepository{}
	ticketRepo := &fakeCommentTicketReader{ticket: &domain.Ticket{ID: ticketID, CreatedBy: ownerID}}
	service := NewCommentService(commentRepo, ticketRepo)

	comment, err := service.CreateComment(context.Background(), ticketID, domain.CreateCommentReq{
		Body:       "I still need help",
		IsInternal: false,
	}, ownerID, "user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if comment == nil || commentRepo.createdComment == nil {
		t.Fatalf("expected comment to be created")
	}
	if commentRepo.createdComment.IsInternal {
		t.Fatalf("expected external comment to be created")
	}
}

func TestCommentServiceUserCannotCreateInternalComment(t *testing.T) {
	ownerID := uuid.New()
	ticketID := uuid.New()
	commentRepo := &fakeCommentRepository{}
	ticketRepo := &fakeCommentTicketReader{ticket: &domain.Ticket{ID: ticketID, CreatedBy: ownerID}}
	service := NewCommentService(commentRepo, ticketRepo)

	_, err := service.CreateComment(context.Background(), ticketID, domain.CreateCommentReq{
		Body:       "Internal note",
		IsInternal: true,
	}, ownerID, "user")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestCommentServiceUserCannotCommentOnAnotherUsersTicket(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()
	ticketID := uuid.New()
	commentRepo := &fakeCommentRepository{}
	ticketRepo := &fakeCommentTicketReader{ticket: &domain.Ticket{ID: ticketID, CreatedBy: ownerID}}
	service := NewCommentService(commentRepo, ticketRepo)

	_, err := service.CreateComment(context.Background(), ticketID, domain.CreateCommentReq{
		Body: "I should not be able to post here",
	}, otherUserID, "user")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestCommentServiceAgentCanCreateInternalComment(t *testing.T) {
	ownerID := uuid.New()
	agentID := uuid.New()
	ticketID := uuid.New()
	commentRepo := &fakeCommentRepository{}
	ticketRepo := &fakeCommentTicketReader{ticket: &domain.Ticket{ID: ticketID, CreatedBy: ownerID, Assignee: &domain.Agent{UserID: agentID}}}
	service := NewCommentService(commentRepo, ticketRepo)

	comment, err := service.CreateComment(context.Background(), ticketID, domain.CreateCommentReq{
		Body:       "Investigating backend logs",
		IsInternal: true,
	}, agentID, "agent")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if comment == nil || !comment.IsInternal {
		t.Fatalf("expected internal comment to be created")
	}
}

func TestCommentServiceListCommentsHidesInternalNotesFromUsers(t *testing.T) {
	ownerID := uuid.New()
	ticketID := uuid.New()
	commentRepo := &fakeCommentRepository{
		listedComments: []domain.Comment{{ID: uuid.New(), TicketID: ticketID}},
	}
	ticketRepo := &fakeCommentTicketReader{ticket: &domain.Ticket{ID: ticketID, CreatedBy: ownerID}}
	service := NewCommentService(commentRepo, ticketRepo)

	_, err := service.ListComments(context.Background(), ticketID, ownerID, "user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if commentRepo.listIncludeInternal {
		t.Fatalf("expected internal comments to be hidden from users")
	}
}

func TestCommentServiceListCommentsAllowsAgentsToViewInternalNotes(t *testing.T) {
	ownerID := uuid.New()
	agentID := uuid.New()
	ticketID := uuid.New()
	commentRepo := &fakeCommentRepository{
		listedComments: []domain.Comment{{ID: uuid.New(), TicketID: ticketID}},
	}
	ticketRepo := &fakeCommentTicketReader{ticket: &domain.Ticket{ID: ticketID, CreatedBy: ownerID, Assignee: &domain.Agent{UserID: agentID}}}
	service := NewCommentService(commentRepo, ticketRepo)

	_, err := service.ListComments(context.Background(), ticketID, agentID, "agent")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !commentRepo.listIncludeInternal {
		t.Fatalf("expected agents to see internal comments")
	}
}

func TestCommentServiceListCommentsForbidsUsersFromOtherTickets(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()
	ticketID := uuid.New()
	commentRepo := &fakeCommentRepository{}
	ticketRepo := &fakeCommentTicketReader{ticket: &domain.Ticket{ID: ticketID, CreatedBy: ownerID}}
	service := NewCommentService(commentRepo, ticketRepo)

	_, err := service.ListComments(context.Background(), ticketID, otherUserID, "user")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestCommentServiceAgentCannotCommentOnUnassignedTicket(t *testing.T) {
	ownerID := uuid.New()
	agentID := uuid.New()
	ticketID := uuid.New()
	commentRepo := &fakeCommentRepository{}
	ticketRepo := &fakeCommentTicketReader{ticket: &domain.Ticket{ID: ticketID, CreatedBy: ownerID}}
	service := NewCommentService(commentRepo, ticketRepo)

	_, err := service.CreateComment(context.Background(), ticketID, domain.CreateCommentReq{
		Body: "I should not be able to post here",
	}, agentID, "agent")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}
