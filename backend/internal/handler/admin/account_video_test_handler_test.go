package admin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type adminVideoTestAccountRepo struct {
	service.AccountRepository
	account *service.Account
}

func (r adminVideoTestAccountRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	if id != r.account.ID {
		return nil, fmt.Errorf("not found")
	}
	return r.account, nil
}

type adminVideoTestHTTP struct {
	service.HTTPUpstream
	requests []*http.Request
	response *http.Response
}

func (u *adminVideoTestHTTP) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	u.requests = append(u.requests, req)
	return u.response, nil
}

func adminVideoTestRouter(t *testing.T, body, contentType string, status int) (*gin.Engine, *adminVideoTestHTTP) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	account := &service.Account{ID: 38, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "test-upstream-secret", "base_url": "https://video.example/v1", "openai_capabilities": []string{"videos"}}}
	upstream := &adminVideoTestHTTP{response: &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(body))}}
	svc := service.NewAccountTestService(adminVideoTestAccountRepo{account: account}, nil, nil, nil, nil, upstream, &config.Config{}, nil)
	handler := &AccountHandler{accountTestService: svc}
	router := gin.New()
	router.GET("/accounts/:id/video-tasks/:task_id", handler.GetVideoTestTask)
	router.GET("/accounts/:id/video-tasks/:task_id/content", handler.GetVideoTestContent)
	return router, upstream
}

func TestAdminVideoTestTaskReturnsFilteredStatus(t *testing.T) {
	router, upstream := adminVideoTestRouter(t, `{"id":"task_123","status":"processing","secret":"test-upstream-secret"}`, "application/json", 200)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/accounts/38/video-tasks/task_123", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"code":0,"message":"success","data":{"id":"task_123","status":"processing"}}`, recorder.Body.String())
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.Len(t, upstream.requests, 1)
	require.Equal(t, http.MethodGet, upstream.requests[0].Method)
}

func TestAdminVideoTestTaskRejectsInvalidPathAndMissingAccount(t *testing.T) {
	for _, path := range []string{"/accounts/no/video-tasks/task_123", "/accounts/0/video-tasks/task_123", "/accounts/38/video-tasks/..", "/accounts/38/video-tasks/task!"} {
		router, upstream := adminVideoTestRouter(t, "", "application/json", 200)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusBadRequest, recorder.Code, path)
		require.Empty(t, upstream.requests)
	}
	router, upstream := adminVideoTestRouter(t, "", "application/json", 200)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/accounts/39/video-tasks/task_123", nil))
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Empty(t, upstream.requests)
}

func TestAdminVideoTestContentStreamsRangeAndFiltersHeaders(t *testing.T) {
	router, upstream := adminVideoTestRouter(t, "mp4-data", "video/mp4", http.StatusPartialContent)
	upstream.response.Header.Set("Content-Range", "bytes 0-7/20")
	upstream.response.Header.Set("Accept-Ranges", "bytes")
	upstream.response.Header.Set("Set-Cookie", "provider-session=test-upstream-secret")
	upstream.response.Header.Set("Authorization", "Bearer test-upstream-secret")
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/accounts/38/video-tasks/task_123/content", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	req.Header.Set("Range", "bytes=0-7")
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusPartialContent, recorder.Code)
	require.Equal(t, "mp4-data", recorder.Body.String())
	require.Equal(t, "video/mp4", recorder.Header().Get("Content-Type"))
	require.Equal(t, "bytes 0-7/20", recorder.Header().Get("Content-Range"))
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.Empty(t, recorder.Header().Get("Set-Cookie"))
	require.Empty(t, recorder.Header().Get("Authorization"))
	require.Equal(t, "Bearer test-upstream-secret", upstream.requests[0].Header.Get("Authorization"))
	require.Equal(t, "bytes=0-7", upstream.requests[0].Header.Get("Range"))
}

func TestAdminVideoTestUpstreamErrorsDoNotLeakCredentials(t *testing.T) {
	for _, suffix := range []string{"", "/content"} {
		router, upstream := adminVideoTestRouter(t, `{"error":"test-upstream-secret rejected"}`, "application/json", http.StatusNotFound)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/accounts/38/video-tasks/task_123"+suffix, nil))
		require.Equal(t, http.StatusNotFound, recorder.Code)
		require.NotContains(t, recorder.Body.String(), "test-upstream-secret")
		require.Len(t, upstream.requests, 1)
	}
}
