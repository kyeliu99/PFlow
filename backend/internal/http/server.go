package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/example/pflow/backend/internal/models"
	"github.com/example/pflow/backend/internal/repository"
	"github.com/example/pflow/backend/internal/service"
)

// Server wraps the gin engine and collaborators needed to handle API requests.
type Server struct {
	Engine   *gin.Engine
	tickets  *repository.TicketRepository
	workflow *service.WorkflowService
}

// NewServer constructs a new API server and registers routes.
func NewServer(repo *repository.TicketRepository, workflow *service.WorkflowService) *Server {
	router := gin.Default()
	srv := &Server{Engine: router, tickets: repo, workflow: workflow}
	srv.registerRoutes()
	return srv
}

func (s *Server) registerRoutes() {
	api := s.Engine.Group("/api")
	api.POST("/tickets", s.createTicket)
	api.GET("/tickets", s.listTickets)
	api.GET("/tickets/:id", s.getTicket)
	api.POST("/tickets/:id/submit", s.submitTicket)
	api.POST("/tickets/:id/decision", s.decision)
}

func (s *Server) createTicket(c *gin.Context) {
	var payload struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		Requester   string `json:"requester" binding:"required"`
		Assignee    string `json:"assignee"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ticket := &models.Ticket{
		Title:       payload.Title,
		Description: payload.Description,
		Requester:   payload.Requester,
		Assignee:    payload.Assignee,
	}

	if err := s.workflow.CreateTicket(c.Request.Context(), ticket); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ticket)
}

func (s *Server) listTickets(c *gin.Context) {
	tickets, err := s.tickets.List(c.Request.Context(), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tickets)
}

func (s *Server) getTicket(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ticket, err := s.tickets.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ticket)
}

func (s *Server) submitTicket(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.workflow.SubmitTicket(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) decision(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload struct {
		Approved bool   `json:"approved"`
		Comment  string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.workflow.RecordDecision(c.Request.Context(), id, payload.Approved, payload.Comment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
