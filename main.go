package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/cert"
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

	// Ensure certificate exists or generate self-signed
	certPath, keyPath, err := cert.EnsureCert(mainCfg.HTTPSCertPath, mainCfg.HTTPSKeyPath)
	if err != nil {
		log.Fatalf("Certificate error: %v", err)
	}
	log.Infof("Using certificate: %s and key: %s", certPath, keyPath)

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
		targets := strategyCfg.URLs
		var counter uint32
		rrCounters[path] = &counter

		switch strategyCfg.Strategy {
		case "round-robin":
			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				target, ok := strategy.RoundRobin(targets, rrCounters[path], banManager)
				if !ok {
					log.Warnf("Round-robin - All backends temporarily banned for %s", path)
					http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
					return
				}
				forward.ForwardRequest(w, r, target)
			})

		case "weighted":
			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				target, ok := strategy.Weighted(targets, banManager)
				if !ok {
					log.Warnf("Weighted - All backends temporarily banned for %s", path)
					http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
					return
				}
				forward.ForwardRequest(w, r, target)
			})

		case "random":
			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				target, ok := strategy.Random(targets, banManager)
				if !ok {
					log.Warnf("Random - All backends temporarily banned for %s", path)
					http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
					return
				}
				forward.ForwardRequest(w, r, target)
			})

		default:
			mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				log.Warnf("Unsupported strategy for %s", path)
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			})
		}
	}

	// Start HTTPS server
	log.Infof("Starting HTTPS server on :%d", mainCfg.Port)
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", mainCfg.Port),
		TLSConfig: tlsConfig,
		Handler:   mux,
	}
	if err := server.ListenAndServeTLS(certPath, keyPath); err != nil {
		log.Fatalf("HTTPS server failed: %v", err)
	}
}
