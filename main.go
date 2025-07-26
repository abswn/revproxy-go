package main

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/abswn/revproxy-go/internal/ban"
	"github.com/abswn/revproxy-go/internal/cert"
	"github.com/abswn/revproxy-go/internal/config"
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

	// TO DO - Set up request router and start server

}
