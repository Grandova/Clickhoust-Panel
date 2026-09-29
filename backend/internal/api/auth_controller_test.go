package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"clickhouse-manager/internal/auth"
	"clickhouse-manager/internal/database"
)

func TestFirstLoginAndIPWhitelist(t *testing.T) {
	dir := t.TempDir()
	if err := database.InitDB(filepath.Join(dir, "panel.db")); err != nil {
		t.Fatal(err)
	}
	db, _ := database.DB.DB()
	defer db.Close()
	if err := auth.InitJWTSecret(filepath.Join(dir, "jwt-secret")); err != nil {
		t.Fatal(err)
	}
	router := SetupRouter()
	token, err := auth.GenerateToken(1, "admin", "admin")
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.RemoteAddr = "192.0.2.10:1234"
		req.Header.Set("X-Forwarded-For", "198.51.100.1")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	if w := request("GET", "/api/v1/auth/me", ""); w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
	if w := request("GET", "/api/v1/settings/security", ""); w.Code != http.StatusForbidden {
		t.Fatalf("first-login gate returned %d", w.Code)
	}
	if w := request("POST", "/api/v1/auth/change-password", `{"old_password":"admin123456","new_password":"admin123456"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("same password accepted: %d", w.Code)
	}
	if w := request("POST", "/api/v1/auth/change-password", `{"old_password":"admin123456","new_password":"local-test-123456"}`); w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
	w := request("GET", "/api/v1/settings/security", "")
	if w.Code != http.StatusOK {
		t.Fatal(w.Body.String())
	}
	var response struct {
		Data struct {
			ClientIP string `json:"client_ip"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.ClientIP != "192.0.2.10" {
		t.Fatalf("trusted spoofed IP: %s", w.Body.String())
	}
	if err := database.DB.Save(&database.PanelSetting{Key: "ip_whitelist", Value: "198.51.100.1"}).Error; err != nil {
		t.Fatal(err)
	}
	if w := request("GET", "/api/v1/auth/me", ""); w.Code != http.StatusForbidden {
		t.Fatalf("spoof bypassed whitelist: %d", w.Code)
	}
}
