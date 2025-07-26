package forward

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

	target := config.URLConfig{
		URL: backend.URL,
	}

	ForwardRequest(rw, req, target)

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

	target := config.URLConfig{
		URL: "://invalid-url",
	}

	ForwardRequest(rw, req, target)

	res := rw.Result()
	if res.StatusCode != http.StatusBadGateway {
		t.Errorf("Expected 502 Bad Gateway for invalid URL, got %d", res.StatusCode)
	}
}

func TestForwardRequest_BackendFailure(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rw := httptest.NewRecorder()

	// Use a non-routable address to simulate backend failure
	target := config.URLConfig{
		URL: "http://127.0.0.1:59999",
	}

	ForwardRequest(rw, req, target)

	res := rw.Result()
	if res.StatusCode != http.StatusBadGateway {
		t.Errorf("Expected 502 Bad Gateway for backend failure, got %d", res.StatusCode)
	}
}
