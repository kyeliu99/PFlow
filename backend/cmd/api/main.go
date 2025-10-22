package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	workflows "github.com/example/pflow/backend/deploy/workflows"
	"github.com/example/pflow/backend/internal/config"
	"github.com/example/pflow/backend/internal/db"
	httpserver "github.com/example/pflow/backend/internal/http"
	"github.com/example/pflow/backend/internal/models"
	"github.com/example/pflow/backend/internal/mq"
	"github.com/example/pflow/backend/internal/repository"
	"github.com/example/pflow/backend/internal/service"
	"github.com/example/pflow/backend/internal/worker"
	"github.com/example/pflow/backend/internal/workflow"
)

func main() {
	cfg := config.Load()

	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	autoMigrate(database)

	var publisher mq.Publisher
	publisher, err = mq.NewRabbitPublisher(cfg.MQURL, cfg.MQTicketExchange)
	if err != nil {
		log.Printf("warning: rabbitmq unavailable (%v), continuing without events", err)
	}
	camundaClient := workflow.NewCamundaClient(cfg.CamundaURL)

	if len(workflows.TicketProcess) > 0 {
		if err := camundaClient.DeployProcess(context.Background(), "ticket-process", workflows.TicketProcess); err != nil {
			log.Printf("deploy workflow failed: %v", err)
		} else {
			log.Println("workflow deployed to camunda")
		}
	}

	ticketRepo := repository.NewTicketRepository(database)
	workflowService := service.NewWorkflowService(database, ticketRepo, camundaClient, publisher, cfg.CamundaProcessKey)
	apiServer := httpserver.NewServer(ticketRepo, workflowService)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go runWorker(ctx, workflowService, camundaClient, cfg)

	srv := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: apiServer.Engine,
	}

	go func() {
		log.Printf("HTTP server listening on %s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown initiated")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	if publisher != nil {
		if closer, ok := publisher.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}
	log.Println("bye")
}

func autoMigrate(db *gorm.DB) {
	if err := db.AutoMigrate(&models.Ticket{}); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}
}

func runWorker(ctx context.Context, svc *service.WorkflowService, camunda *workflow.CamundaClient, cfg config.Config) {
	worker := worker.NewExternalWorker("ticket-processing", svc, camunda, 5*time.Second, cfg.WorkerLockDuration)
	worker.Run(ctx)
}

func init() {
	if mode := os.Getenv("GIN_MODE"); mode == "" {
		gin.SetMode(gin.ReleaseMode)
	}
}
