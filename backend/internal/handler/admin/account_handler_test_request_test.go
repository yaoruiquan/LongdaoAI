package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountHandlerTestRejectsMalformedVideoRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := &AccountHandler{}
	router.POST("/accounts/:id/test", handler.Test)
	for _, body := range []string{
		`{"mode":"video","video":`,
		`{"mode":"video","video":{"seconds":"6"}}`,
		`{"mode":"video","video":{"seconds":1.5}}`,
		`{"mode":"video","video":{"size":720}}`,
	} {
		t.Run(body, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/accounts/38/test", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Contains(t, recorder.Body.String(), "Invalid account test request")
		})
	}
}
