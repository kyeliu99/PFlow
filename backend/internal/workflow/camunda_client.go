package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"
)

// CamundaClient exposes a small subset of Camunda REST operations used by the application.
type CamundaClient struct {
	baseURL string
	client  *http.Client
}

// NewCamundaClient constructs a client targeting the provided base URL.
func NewCamundaClient(baseURL string) *CamundaClient {
	return &CamundaClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

// DeployProcess deploys a BPMN definition to Camunda.
func (c *CamundaClient) DeployProcess(ctx context.Context, name string, bpmn []byte) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("data", fmt.Sprintf("%s.bpmn", name))
	if err != nil {
		return err
	}
	if _, err := part.Write(bpmn); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/deployment/create", c.baseURL), body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("camunda deploy failed: %s", resp.Status)
	}
	return nil
}

// StartProcessInstance starts a process by key and business key.
func (c *CamundaClient) StartProcessInstance(ctx context.Context, key, businessKey string, variables map[string]any) (string, error) {
	payload := map[string]any{
		"variables":   wrapVariables(variables),
		"businessKey": businessKey,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/process-definition/key/%s/start", c.baseURL, key), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("failed to start process: %s", resp.Status)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.ID, nil
}

// FetchAndLockExternalTasks pulls external tasks for the given topic.
func (c *CamundaClient) FetchAndLockExternalTasks(ctx context.Context, workerID, topic string, lockDuration time.Duration) ([]ExternalTask, error) {
	payload := map[string]any{
		"workerId":    workerID,
		"maxTasks":    5,
		"usePriority": true,
		"topics": []map[string]any{{
			"topicName":    topic,
			"lockDuration": int(lockDuration.Milliseconds()),
		}},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/external-task/fetchAndLock", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetchAndLock failed: %s", resp.Status)
	}
	var tasks []ExternalTask
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// CompleteExternalTask completes a locked external task with optional variables.
func (c *CamundaClient) CompleteExternalTask(ctx context.Context, workerID, taskID string, variables map[string]any) error {
	payload := map[string]any{
		"workerId":  workerID,
		"variables": wrapVariables(variables),
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/external-task/%s/complete", c.baseURL, taskID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("complete failed: %s", resp.Status)
	}
	return nil
}

// ExternalTask mirrors the Camunda response.
type ExternalTask struct {
	ID           string `json:"id"`
	ProcessID    string `json:"processInstanceId"`
	ActivityID   string `json:"activityId"`
	TopicName    string `json:"topicName"`
	BusinessKey  string `json:"businessKey"`
	VariablesRaw map[string]struct {
		Type  string      `json:"type"`
		Value interface{} `json:"value"`
	} `json:"variables"`
}

func wrapVariables(vars map[string]any) map[string]any {
	if vars == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(vars))
	for k, v := range vars {
		out[k] = map[string]any{
			"value": v,
		}
	}
	return out
}
