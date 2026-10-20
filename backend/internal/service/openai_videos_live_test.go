package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Opt-in only: this test creates one billable task unless A6API_VIDEO_TASK_ID is set.
// Credentials are read from the environment and are never saved in the artifacts.
func TestOpenAIVideosA6Live(t *testing.T) {
	key := os.Getenv("A6API_VIDEO_TEST_KEY")
	if key == "" {
		t.Skip("set A6API_VIDEO_TEST_KEY to run the paid video integration test")
	}
	output := os.Getenv("A6API_VIDEO_TEST_OUTPUT")
	if output == "" {
		output = t.TempDir()
	}
	require.NoError(t, os.MkdirAll(output, 0755))
	client := &http.Client{Timeout: 180 * time.Second}
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": key, "base_url": "https://api.a6api.com"}}
	apiKey := &APIKey{ID: 1}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, cache: &videoTestCache{}, httpUpstream: videoTestUpstream{do: client.Do}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	taskID := os.Getenv("A6API_VIDEO_TASK_ID")
	if taskID == "" {
		body := []byte(`{"model":"minimax-h3","prompt":"A small paper boat gently floating on a calm pond, soft daylight, static camera, no people, no text.","seconds":"6","size":"1280x720"}`)
		request, err := ParseOpenAIVideoRequest(body, "application/json")
		require.NoError(t, err)
		c, recorder := videoTestContext(http.MethodPost, "/v1/videos", body, request.ContentType)
		result, err := svc.ForwardVideo(ctx, c, account, apiKey, request, "", false, "")
		response := strings.ReplaceAll(recorder.Body.String(), key, "[REDACTED]")
		require.NoError(t, os.WriteFile(filepath.Join(output, "create.json"), []byte(response), 0644))
		t.Logf("create HTTP %d: %s", recorder.Code, response)
		require.NoError(t, err)
		require.Less(t, recorder.Code, 300)
		taskID = result.ResponseID
		require.NoError(t, os.WriteFile(filepath.Join(output, "request.json"), body, 0644))
	}
	require.NotEmpty(t, taskID)
	t.Logf("task ID: %s", taskID)
	for {
		c, recorder := videoTestContext(http.MethodGet, "/v1/videos/"+taskID, nil, "")
		result, err := svc.ForwardVideo(ctx, c, account, apiKey, nil, taskID, false, "")
		response := strings.ReplaceAll(recorder.Body.String(), key, "[REDACTED]")
		require.NoError(t, err, "status HTTP %d: %s", recorder.Code, response)
		require.Equal(t, 0, result.VideoCount, "polling must not bill again")
		require.NoError(t, os.WriteFile(filepath.Join(output, "status.json"), []byte(response), 0644))
		status := gjson.Get(response, "status").String()
		t.Logf("task %s status=%s progress=%s", taskID, status, gjson.Get(response, "progress").String())
		switch status {
		case "completed", "succeeded", "success":
			c, download := videoTestContext(http.MethodGet, "/v1/videos/"+taskID+"/content", nil, "")
			_, err := svc.ForwardVideo(ctx, c, account, apiKey, nil, taskID, true, "")
			require.NoError(t, err, "download HTTP %d: %s", download.Code, strings.ReplaceAll(download.Body.String(), key, "[REDACTED]"))
			require.Greater(t, download.Body.Len(), 1024)
			require.True(t, bytes.Contains(download.Body.Bytes()[:min(64, download.Body.Len())], []byte("ftyp")), "expected an MP4 file")
			require.NoError(t, os.WriteFile(filepath.Join(output, "minimax-h3.mp4"), download.Body.Bytes(), 0644))
			metadata, _ := json.MarshalIndent(map[string]any{"model": "minimax-h3", "task_id": taskID, "status": status, "content_type": download.Header().Get("Content-Type"), "video_bytes": download.Body.Len(), "tested_at": time.Now().UTC().Format(time.RFC3339)}, "", "  ")
			require.NoError(t, os.WriteFile(filepath.Join(output, "result.json"), metadata, 0644))
			t.Logf("downloaded %d bytes to %s", download.Body.Len(), output)
			return
		case "failed", "cancelled", "canceled", "expired":
			t.Fatal(fmt.Sprintf("video task %s: %s", taskID, response))
		}
		select {
		case <-ctx.Done():
			t.Fatal("video task is still pending; resume with A6API_VIDEO_TASK_ID=" + taskID)
		case <-time.After(10 * time.Second):
		}
	}
}
