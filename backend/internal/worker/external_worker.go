package worker

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/example/pflow/backend/internal/service"
	"github.com/example/pflow/backend/internal/workflow"
)

// ExternalWorker continuously polls Camunda for external tasks on a topic and delegates to the service.
type ExternalWorker struct {
	id       string
	topic    string
	service  *service.WorkflowService
	camunda  *workflow.CamundaClient
	interval time.Duration
	lock     time.Duration
}

// NewExternalWorker creates the worker with random identifier.
func NewExternalWorker(topic string, svc *service.WorkflowService, camunda *workflow.CamundaClient, interval, lock time.Duration) *ExternalWorker {
	return &ExternalWorker{
		id:       uuid.New().String(),
		topic:    topic,
		service:  svc,
		camunda:  camunda,
		interval: interval,
		lock:     lock,
	}
}

// Run starts the polling loop and should be launched in its own goroutine.
func (w *ExternalWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("external worker shutting down")
			return
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *ExternalWorker) poll(ctx context.Context) {
	tasks, err := w.camunda.FetchAndLockExternalTasks(ctx, w.id, w.topic, w.lock)
	if err != nil {
		log.Printf("fetch external tasks error: %v", err)
		return
	}
	for _, task := range tasks {
		if err := w.service.HandleExternalTask(ctx, task); err != nil {
			log.Printf("handle external task %s failed: %v", task.ID, err)
			continue
		}
		if err := w.camunda.CompleteExternalTask(ctx, w.id, task.ID, map[string]any{"handledAt": time.Now().UTC().Format(time.RFC3339)}); err != nil {
			log.Printf("complete external task %s failed: %v", task.ID, err)
		}
	}
}
