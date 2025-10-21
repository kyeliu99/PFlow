package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TicketStatus describes the life-cycle state of a ticket in the workflow.
type TicketStatus string

const (
	TicketStatusDraft      TicketStatus = "draft"
	TicketStatusSubmitted  TicketStatus = "submitted"
	TicketStatusApproved   TicketStatus = "approved"
	TicketStatusRejected   TicketStatus = "rejected"
	TicketStatusProcessing TicketStatus = "processing"
	TicketStatusCompleted  TicketStatus = "completed"
)

// Ticket represents a work order entity persisted in Postgres and mirrored in Camunda.
type Ticket struct {
	ID                uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	Title             string       `json:"title"`
	Description       string       `json:"description"`
	Requester         string       `json:"requester"`
	Assignee          string       `json:"assignee"`
	Status            TicketStatus `json:"status"`
	ProcessInstanceID string       `json:"processInstanceId"`
	CreatedAt         time.Time    `json:"createdAt"`
	UpdatedAt         time.Time    `json:"updatedAt"`
}

// BeforeCreate is a GORM hook that populates the primary key.
func (t *Ticket) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.Status == "" {
		t.Status = TicketStatusDraft
	}
	return nil
}
