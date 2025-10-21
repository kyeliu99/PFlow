package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/example/pflow/backend/internal/models"
)

// TicketRepository provides persistence access for Ticket entities.
type TicketRepository struct {
	db *gorm.DB
}

// NewTicketRepository constructs a repository using the provided gorm DB.
func NewTicketRepository(db *gorm.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

// Create persists the ticket instance.
func (r *TicketRepository) Create(ctx context.Context, ticket *models.Ticket) error {
	return errors.WithStack(r.db.WithContext(ctx).Create(ticket).Error)
}

// Update persists the modified ticket.
func (r *TicketRepository) Update(ctx context.Context, ticket *models.Ticket) error {
	return errors.WithStack(r.db.WithContext(ctx).Save(ticket).Error)
}

// FindByID returns the ticket by id.
func (r *TicketRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Ticket, error) {
	var ticket models.Ticket
	if err := r.db.WithContext(ctx).First(&ticket, "id = ?", id).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return &ticket, nil
}

// List returns all tickets ordered by creation time descending.
func (r *TicketRepository) List(ctx context.Context, limit int) ([]models.Ticket, error) {
	if limit <= 0 {
		limit = 50
	}
	var tickets []models.Ticket
	err := r.db.WithContext(ctx).Order("created_at desc").Limit(limit).Find(&tickets).Error
	return tickets, errors.WithStack(err)
}
