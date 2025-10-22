package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/example/pflow/backend/internal/models"
	"github.com/example/pflow/backend/internal/mq"
	"github.com/example/pflow/backend/internal/repository"
	"github.com/example/pflow/backend/internal/workflow"
)

// WorkflowService contains business logic for bridging persistence and the workflow engine.
type WorkflowService struct {
	db         *gorm.DB
	tickets    *repository.TicketRepository
	camunda    *workflow.CamundaClient
	mq         mq.Publisher
	processKey string
}

// NewWorkflowService builds a service with dependencies.
func NewWorkflowService(db *gorm.DB, repo *repository.TicketRepository, camunda *workflow.CamundaClient, mq mq.Publisher, processKey string) *WorkflowService {
	return &WorkflowService{db: db, tickets: repo, camunda: camunda, mq: mq, processKey: processKey}
}

// CreateTicket persists a new ticket, publishes an event and keeps it in draft status.
func (s *WorkflowService) CreateTicket(ctx context.Context, ticket *models.Ticket) error {
	ticket.Status = models.TicketStatusDraft
	if err := s.tickets.Create(ctx, ticket); err != nil {
		return err
	}
	if err := s.publishEvent(ctx, "ticket.created", ticket); err != nil {
		log.Printf("publish ticket.created failed: %v", err)
	}
	return nil
}

// SubmitTicket transitions a ticket into the workflow and starts a process instance.
func (s *WorkflowService) SubmitTicket(ctx context.Context, ticketID uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ticket, err := s.tickets.FindByID(ctx, ticketID)
		if err != nil {
			return err
		}
		if ticket.Status != models.TicketStatusDraft && ticket.Status != models.TicketStatusRejected {
			return errors.Errorf("ticket %s cannot be submitted from status %s", ticket.ID, ticket.Status)
		}
		pid, err := s.camunda.StartProcessInstance(ctx, s.processKey, ticket.ID.String(), map[string]any{
			"requester": ticket.Requester,
			"title":     ticket.Title,
		})
		if err != nil {
			return err
		}
		ticket.ProcessInstanceID = pid
		ticket.Status = models.TicketStatusSubmitted
		if err := s.tickets.Update(ctx, ticket); err != nil {
			return err
		}
		return s.publishEvent(ctx, "ticket.submitted", ticket)
	})
}

// RecordDecision records manager decision and advances the process via external task completion.
func (s *WorkflowService) RecordDecision(ctx context.Context, ticketID uuid.UUID, approved bool, comment string) error {
	ticket, err := s.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return err
	}
	if ticket.Status != models.TicketStatusSubmitted && ticket.Status != models.TicketStatusProcessing {
		return errors.Errorf("ticket %s is not awaiting decision", ticket.ID)
	}

	if approved {
		ticket.Status = models.TicketStatusApproved
	} else {
		ticket.Status = models.TicketStatusRejected
	}
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return err
	}
	if err := s.publishEvent(ctx, "ticket.decision", ticket); err != nil {
		log.Printf("publish ticket.decision failed: %v", err)
	}
	return nil
}

// CompleteProcessing marks the ticket as completed after asynchronous processing.
func (s *WorkflowService) CompleteProcessing(ctx context.Context, ticketID uuid.UUID) error {
	ticket, err := s.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return err
	}
	ticket.Status = models.TicketStatusCompleted
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return err
	}
	return s.publishEvent(ctx, "ticket.completed", ticket)
}

func (s *WorkflowService) publishEvent(ctx context.Context, event string, ticket *models.Ticket) error {
	if s.mq == nil {
		return nil
	}
	payload := map[string]any{
		"event":      event,
		"ticketId":   ticket.ID.String(),
		"status":     ticket.Status,
		"processId":  ticket.ProcessInstanceID,
		"title":      ticket.Title,
		"requester":  ticket.Requester,
		"assignee":   ticket.Assignee,
		"occurredAt": time.Now().UTC().Format(time.RFC3339),
	}
	return s.mq.Publish(ctx, event, payload)
}

// HandleExternalTask is invoked by worker when asynchronous steps are completed.
func (s *WorkflowService) HandleExternalTask(ctx context.Context, task workflow.ExternalTask) error {
	switch task.ActivityID {
	case "ServiceTask_ProcessTicket":
		ticketID, err := uuid.Parse(task.BusinessKey)
		if err != nil {
			return errors.Wrap(err, "invalid business key")
		}
		log.Printf("processing external task for ticket %s", ticketID)
		if err := s.transitionToProcessing(ctx, ticketID); err != nil {
			return err
		}
		return s.publishEvent(ctx, "ticket.processing", map[string]any{
			"ticketId": ticketID.String(),
			"activity": task.ActivityID,
		})
	default:
		return fmt.Errorf("unhandled activity %s", task.ActivityID)
	}
}

func (s *WorkflowService) transitionToProcessing(ctx context.Context, ticketID uuid.UUID) error {
	ticket, err := s.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return err
	}
	ticket.Status = models.TicketStatusProcessing
	return s.tickets.Update(ctx, ticket)
}
