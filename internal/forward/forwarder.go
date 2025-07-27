package forward

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/config"
	"golang.org/x/net/proxy"
)

// ForwardRequest forwards the request to the given target URL.
func ForwardRequest(w http.ResponseWriter, r *http.Request, target config.URLConfig, banRules []config.BanRuleClean, bm *ban.BanManager) error {
	// Parse the target URL to ensure it's valid
	parsedURL, err := url.Parse(target.URL)
	if err != nil {
		log.Errorf("Invalid target URL: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return err
	}

	// Create outbound request using r.Context() so that client disconnection cancels backend request
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, parsedURL.String(), r.Body)
	if err != nil {
		log.Errorf("Failed to create proxy request: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return err
	}

	// Clone headers from the original request to the new one
	proxyReq.Header = r.Header.Clone()

	// 60 seconds timeout for the requests
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// If SOCKS5 proxy is specified, configure a custom Transport
	if target.Socks5 != "" {
		var auth *proxy.Auth
		if target.Username != "" || target.Password != "" {
			auth = &proxy.Auth{
				User:     target.Username,
				Password: target.Password,
			}
		}

		// Create a SOCKS5 dialer
		dialer, err := proxy.SOCKS5("tcp", target.Socks5, auth, &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 10 * time.Second,
		})
		if err != nil {
			log.Errorf("Failed to create SOCKS5 dialer: %v", err)
			// return error to prevent unexpected routing
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return err
		}
		// Override the default HTTP transport to route through SOCKS5
		client.Transport = &http.Transport{
			DialContext: dialer.(proxy.ContextDialer).DialContext,
		}
	}

	// Send the request to the backend URL
	log.Infof("Forwarding %s request for %s to backend %s", r.Method, r.URL.Path, parsedURL)
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Errorf("Request to backend failed: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return err
	}
	defer resp.Body.Close()

	// Copy all headers from the backend response to the client
	for k, v := range resp.Header {
		w.Header()[k] = v
	}

	// Set the backend response status code
	w.WriteHeader(resp.StatusCode)

	// Stream the backend response body directly to the client
	// _, copyErr := io.Copy(w, resp.Body)

	// Use TeeReader to hold a copy of the response object for analysis
	var bodyBuffer bytes.Buffer
	tee := io.TeeReader(resp.Body, &bodyBuffer)

	// Write response to client
	_, copyErr := io.Copy(w, tee)
	if copyErr != nil {
		log.Warnf("Failed to copy response body: %v", copyErr)
	}

	// Analyze response
	bodyStr := strings.ToLower(bodyBuffer.String())
	if len(bodyStr) > 200 {
		bodyStr = bodyStr[:200]
	}
	statusCodeStr := strconv.Itoa(resp.StatusCode)
	statusText := strings.ToLower(resp.Status)
	shouldBan := false
	banDuration := 0
	for _, rule := range banRules {
		word := strings.ToLower(rule.Match)
		if word == statusCodeStr ||
			strings.Contains(statusText, word) ||
			strings.Contains(bodyStr, word) {
			shouldBan = true
			banDuration = rule.Duration
			break
		}
	}
	if shouldBan {
		log.Infof("Banning URL %s %s %s", target.URL, resp.Status, bodyStr)
		bm.BanURL(target.URL, time.Duration(banDuration)*time.Second)
	}

	return nil
}
