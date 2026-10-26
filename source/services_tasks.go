package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	cloudtaskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
)

func enqueueLeadProcessingTask(ctx context.Context, cfg Config, leadID string) error {
	if strings.TrimSpace(cfg.CloudTasksProjectID) == "" {
		return fmt.Errorf("CLOUD_TASKS_PROJECT_ID is empty")
	}
	if strings.TrimSpace(cfg.InternalProcessLeadURL) == "" {
		return fmt.Errorf("INTERNAL_PROCESS_LEAD_URL is empty")
	}
	if strings.TrimSpace(cfg.InternalTaskToken) == "" {
		return fmt.Errorf("INTERNAL_TASK_TOKEN is empty")
	}

	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("create cloud tasks client: %w", err)
	}
	defer client.Close()

	payload, err := json.Marshal(LeadTaskPayload{LeadID: leadID})
	if err != nil {
		return fmt.Errorf("marshal lead task payload: %w", err)
	}

	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", cfg.CloudTasksProjectID, cfg.CloudTasksLocation, cfg.CloudTasksQueue)
	request := &cloudtaskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &cloudtaskspb.Task{
			ScheduleTime: nil,
			MessageType: &cloudtaskspb.Task_HttpRequest{
				HttpRequest: &cloudtaskspb.HttpRequest{
					HttpMethod: cloudtaskspb.HttpMethod_POST,
					Url:        cfg.InternalProcessLeadURL,
					Headers: map[string]string{
						"Content-Type":          "application/json",
						"X-Internal-Task-Token": cfg.InternalTaskToken,
					},
					Body: payload,
				},
			},
		},
	}

	if strings.TrimSpace(cfg.TasksServiceAccountEmail) != "" {
		httpReq := request.Task.GetHttpRequest()
		httpReq.AuthorizationHeader = &cloudtaskspb.HttpRequest_OidcToken{
			OidcToken: &cloudtaskspb.OidcToken{
				ServiceAccountEmail: cfg.TasksServiceAccountEmail,
				Audience:            cfg.InternalProcessLeadURL,
			},
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	_, err = client.CreateTask(ctx, request)
	if err != nil {
		return fmt.Errorf("create cloud task: %w", err)
	}

	return nil
}

func isAuthorizedInternalTaskRequest(cfg Config, r *http.Request) bool {
	expected := strings.TrimSpace(cfg.InternalTaskToken)
	actual := strings.TrimSpace(r.Header.Get("X-Internal-Task-Token"))

	if expected == "" {
		return false
	}

	if actual == "" {
		return false
	}

	return actual == expected
}
