package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/config"
	"github.com/abswn/revproxy-go/internal/forward"
	"github.com/abswn/revproxy-go/internal/strategy"
)

func main() {
	// Load main config
	mainCfg, err := config.LoadMainConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load main config: %v", err)
	}
	// Setup logger using logrus
	level, err := log.ParseLevel(mainCfg.Log.Level)
	if err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}
	log.SetLevel(level)
	if mainCfg.Log.Output == "stdout" || mainCfg.Log.Output == "" {
		log.SetOutput(os.Stdout)
	} else {
		f, err := os.OpenFile(mainCfg.Log.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		log.SetOutput(f)
		defer f.Close()
	}
	if mainCfg.Log.Format == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
	}
	log.Info("Logger initialized.")

	// Load enabled endpoints
	endpointsMap, err := config.LoadEnabledEndpointsMap("configs/endpoints")
	if err != nil {
		log.Fatalf("Failed to load endpoint configs: %v", err)
	}
	log.Infof("Loaded %d enabled endpoint config(s)", len(endpointsMap))

	// Initialize the BanManager - temporarily bans backend URLs that hit RPS limit, etc.
	banManager := ban.NewManager()
	banManager.StartEvictionLoop(5 * time.Second) // Check if it is time to re-add the banned URLs

	// Initialize round-robin counters per client endpoint
	rrCounters := make(map[string]*uint32)

	// Create the request multiplexer - matches incoming requests to handlers
	mux := http.NewServeMux()

	for path, strategyCfg := range endpointsMap {
		log.Debugf("Loaded raw endpoint config keys: %v", reflect.ValueOf(endpointsMap).MapKeys())

		targets := strategyCfg.URLs
		// Initialize counter for round-robin
		if strategyCfg.Strategy == "round-robin" {
			rrCounters[path] = new(uint32)
		}
		// Register HTTP handler for each path
		mux.HandleFunc(path, recoveryMiddleware(func(w http.ResponseWriter, r *http.Request) {
			var (
				target config.URLConfig
				ok     bool
			)
			log.Debugf("Registered handler for path: %s", path)
			// Determine strategy
			switch strategyCfg.Strategy {
			case "round-robin":
				target, ok = strategy.RoundRobin(targets, rrCounters[path], banManager)
			case "weighted":
				target, ok = strategy.Weighted(targets, banManager)
			case "random":
				target, ok = strategy.Random(targets, banManager)
			default:
				// Unknown strategy, respond with 503
				log.Warnf("Unsupported strategy '%s' for path %s", strategyCfg.Strategy, path)
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
			// If no usable backends are available
			if !ok {
				log.Warnf("%s - All backends temporarily banned for %s", strategyCfg.Strategy, path)
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
			// Forward request to selected backend
			forward.ForwardRequest(w, r, target)
		}))
	}

	// Start HTTPS server
	log.Infof("Starting server on port :%d", mainCfg.Port)
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", mainCfg.Port),
		TLSConfig: tlsConfig,
		Handler:   mux,
	}
	certExists := func() bool { _, err := os.Stat(mainCfg.HTTPSCertPath); return err == nil }()
	keyExists := func() bool { _, err := os.Stat(mainCfg.HTTPSKeyPath); return err == nil }()
	if certExists && keyExists {
		log.Infof("TLS certificates found. Starting HTTPS server")
		if err := server.ListenAndServeTLS(mainCfg.HTTPSCertPath, mainCfg.HTTPSKeyPath); err != nil {
			log.Fatalf("HTTPS server failed: %v", err)
		}
	} else {
		log.Warnf("TLS certificates not found. Falling back to HTTP.")
		server.TLSConfig = nil
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}
}

// recoveryMiddleware recovers from panics in HTTP handlers and responds with 500 Internal Server Error.
func recoveryMiddleware(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Errorf("Panic recovered in handler for %s: %v", r.URL.Path, rec)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		fn(w, r)
	}
}
