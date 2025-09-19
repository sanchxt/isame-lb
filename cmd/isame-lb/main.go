package main

import (
	"flag"
	"log"
	"os"

	"github.com/sanchxt/isame-lb/internal/config"
	"github.com/sanchxt/isame-lb/internal/server"
)

func main() {
	// cli flags
	var configFile string
	flag.StringVar(&configFile, "config", "configs/dev.yaml", "Path to configuration file")
	flag.Parse()

	log.Println("Isame Load Balancer starting...")

	// load config
	cfg, err := config.LoadConfigWithDefaults(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// if upstreams, validate config
	if len(cfg.Upstreams) > 0 {
		if err := cfg.Validate(); err != nil {
			log.Fatalf("Configuration validation failed: %v", err)
		}
	} else {
		log.Println("Warning: No upstreams configured. Load balancer will return 503 for all requests.")
	}

	log.Printf("Configuration loaded: %s v%s", cfg.Service, cfg.Version)
	log.Printf("Upstreams: %d, Health checks: %v, Metrics: %v",
		len(cfg.Upstreams), cfg.Health.Enabled, cfg.Metrics.Enabled)

	// create and start the server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// start the server (blocks until shutdown)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Isame Load Balancer stopped")
	os.Exit(0)
}
