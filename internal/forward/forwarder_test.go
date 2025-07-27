package forward

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/config"
)

func TestForwardRequest_Success(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from backend"))
	}))
	defer backend.Close()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rw := httptest.NewRecorder()

	target := config.URLConfig{URL: backend.URL}
	bm := ban.NewManager()

	err := ForwardRequest(rw, req, target, []config.BanRuleClean{}, bm)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	res := rw.Result()
	body, _ := io.ReadAll(res.Body)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", res.StatusCode)
	}
	if !strings.Contains(string(body), "Hello from backend") {
		t.Errorf("Unexpected body: %s", string(body))
	}
	if res.Header.Get("X-Test-Header") != "test-value" {
		t.Errorf("Missing or incorrect header: %s", res.Header.Get("X-Test-Header"))
	}
}

func TestForwardRequest_InvalidURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rw := httptest.NewRecorder()

	target := config.URLConfig{URL: "://invalid-url"}
	bm := ban.NewManager()

	err := ForwardRequest(rw, req, target, []config.BanRuleClean{}, bm)
	if err == nil {
		t.Error("Expected error for invalid URL but got nil")
	}

	res := rw.Result()
	if res.StatusCode != http.StatusBadGateway {
		t.Errorf("Expected 502 Bad Gateway, got %d", res.StatusCode)
	}
}

func TestForwardRequest_BackendFailure(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rw := httptest.NewRecorder()

	target := config.URLConfig{URL: "http://127.0.0.1:59999"} // non-routable
	bm := ban.NewManager()

	err := ForwardRequest(rw, req, target, []config.BanRuleClean{}, bm)
	if err == nil {
		t.Error("Expected error for backend failure but got nil")
	}

	res := rw.Result()
	if res.StatusCode != http.StatusBadGateway {
		t.Errorf("Expected 502 Bad Gateway, got %d", res.StatusCode)
	}
}

func TestForwardRequest_BanTriggeredByStatusCode(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // 418
		w.Write([]byte("I'm a teapot"))
	}))
	defer backend.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rw := httptest.NewRecorder()

	rules := []config.BanRuleClean{
		{Match: "418", Duration: 5},
	}
	bm := ban.NewManager()

	err := ForwardRequest(rw, req, config.URLConfig{URL: backend.URL}, rules, bm)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !bm.IsBanned(backend.URL) {
		t.Errorf("Expected backend URL to be banned")
	}
}

func TestForwardRequest_BanTriggeredByStatusText(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request error", http.StatusBadRequest) // 400 Bad Request
	}))
	defer backend.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rw := httptest.NewRecorder()

	rules := []config.BanRuleClean{
		{Match: "bad request", Duration: 10},
	}
	bm := ban.NewManager()

	err := ForwardRequest(rw, req, config.URLConfig{URL: backend.URL}, rules, bm)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !bm.IsBanned(backend.URL) {
		t.Errorf("Expected URL to be banned by status text")
	}
}

func TestForwardRequest_BanTriggeredByBody(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("temporary backend overload"))
	}))
	defer backend.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rw := httptest.NewRecorder()

	rules := []config.BanRuleClean{
		{Match: "overload", Duration: 8},
	}
	bm := ban.NewManager()

	err := ForwardRequest(rw, req, config.URLConfig{URL: backend.URL}, rules, bm)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !bm.IsBanned(backend.URL) {
		t.Errorf("Expected ban triggered by body content")
	}
}

func TestForwardRequest_BanNotTriggered(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("everything fine"))
	}))
	defer backend.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rw := httptest.NewRecorder()

	rules := []config.BanRuleClean{
		{Match: "error", Duration: 5},
	}
	bm := ban.NewManager()

	err := ForwardRequest(rw, req, config.URLConfig{URL: backend.URL}, rules, bm)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if bm.IsBanned(backend.URL) {
		t.Errorf("URL should not have been banned")
	}
}

func TestForwardRequest_LongBodyTruncatedBeforeMatch(t *testing.T) {
	longBody := strings.Repeat("A", 300) + "ERROR"
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(longBody))
	}))
	defer backend.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rw := httptest.NewRecorder()

	rules := []config.BanRuleClean{
		{Match: "ERROR", Duration: 5},
	}
	bm := ban.NewManager()

	err := ForwardRequest(rw, req, config.URLConfig{URL: backend.URL}, rules, bm)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if bm.IsBanned(backend.URL) {
		t.Errorf("URL should not be banned because match is outside 200 bytes")
	}
}
