package utils

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestClickHousePermissionSuggestion(t *testing.T) {
	for _, detail := range []string{
		"code: 497, message: default: Not enough privileges. Missing permissions: SHOW NAMED COLLECTIONS SECRETS ON *",
		"Code: 497. DB::Exception: yoshino: Not enough privileges. CREATE TABLE ON default.online_ip (ACCESS_DENIED)",
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		ClickHouseError(c, "授权失败", errors.New(detail), "")
		var response Response
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(response.Suggestion, "GRANT OPTION") || strings.Contains(response.Suggestion, "网络") {
			t.Fatalf("wrong suggestion: %s", response.Suggestion)
		}
		if response.ErrorDetail != detail {
			t.Fatal("original error lost")
		}
	}
}
