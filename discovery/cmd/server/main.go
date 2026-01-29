// Package main is the entry point for the discovery server.
package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"

	"github.com/agntcy/dir/discovery/pkg/config"
	"github.com/agntcy/dir/discovery/pkg/storage"
	"github.com/agntcy/dir/discovery/pkg/types"
)

var store types.StoreReader

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	if cfg.Storage.StorageType != "etcd" {
		log.Fatalf("Unsupported storage type: %s", cfg.Storage.StorageType)
	}

	log.Println("============================================================")
	log.Println("Discovery Server (Go)")
	log.Println("============================================================")
	log.Printf("Storage: etcd @ %s", cfg.Storage.Etcd.Endpoints()[0])
	log.Printf("Workloads prefix: %s", cfg.Storage.Etcd.WorkloadsPrefix)
	log.Printf("Server: %s", cfg.Server.Addr())
	log.Println("============================================================")

	// Initialize storage
	store, err = storage.NewReader(cfg.Storage)
	if err != nil {
		log.Fatalf("Failed to connect to storage: %v", err)
	}
	defer store.Close()

	// Setup router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health", "/healthz"},
	}))

	// Routes
	router.GET("/healthz", healthHandler)
	router.GET("/discover", discoverHandler)
	router.GET("/workloads", workloadsHandler)
	router.GET("/workload/:id", workloadHandler)

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on %s", cfg.Server.Addr())
		if err := router.Run(cfg.Server.Addr()); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down server...")
}

// healthHandler returns a health check response.
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "discovery-server",
	})
}

// discoverHandler discovers reachable workloads from a source.
func discoverHandler(c *gin.Context) {
	from := c.Query("from")
	if from == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'from' parameter"})
		return
	}

	// Find reachable workloads
	result, err := store.FindReachable(from)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// workloadsHandler returns all registered workloads.
func workloadsHandler(c *gin.Context) {
	runtime := c.Query("runtime")
	workloads := store.List(types.RuntimeType(runtime), nil)

	c.JSON(http.StatusOK, gin.H{
		"workloads": workloads,
		"count":     len(workloads),
	})
}

// workloadHandler returns a single workload by ID.
func workloadHandler(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing workload ID"})
		return
	}

	workload, err := store.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workload not found"})
		return
	}

	c.JSON(http.StatusOK, workload)
}
