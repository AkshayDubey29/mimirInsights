package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	github.com/akshaydubey29/mimirInsights/pkg/api"
	github.com/akshaydubey29/mimirInsights/pkg/config"
	github.com/akshaydubey29/mimirInsights/pkg/discovery"
	"github.com/akshaydubey29/mimirInsights/pkg/drift"
	"github.com/akshaydubey29/mimirInsights/pkg/k8s"
	github.com/akshaydubey29/mimirInsights/pkg/limits"
	"github.com/akshaydubey29/mimirInsights/pkg/llm"
	github.com/akshaydubey29/mimirInsights/pkg/metrics"
	"github.com/akshaydubey29/mimirInsights/pkg/planner"
	github.com/gin-gonic/gin"
	github.com/sirupsen/logrus"
	github.com/spf13/viper"
)

func main() {
	// Initialize configuration
	if err := config.Init(); err != nil {
		logrus.Fatalf("Failed to initialize config: %v", err)
	}

	// Set log level
	logLevel := viper.GetString("log.level")
	if level, err := logrus.ParseLevel(logLevel); err == nil {
		logrus.SetLevel(level)
	}

	logrus.Info("Starting MimirInsights server...")

	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		logrus.Fatalf("Failed to initialize Kubernetes client: %v", err)
	}

	// Initialize components
	discoveryEngine := discovery.NewEngine(k8sClient)
	metricsClient := metrics.NewClient()
	limitsAnalyzer := limits.NewAnalyzer(metricsClient)
	driftDetector := drift.NewDetector(k8sClient, discoveryEngine)
	capacityPlanner := planner.NewPlanner(metricsClient, limitsAnalyzer)
	llmAssistant := llm.NewAssistant()

	// Create API server
	server := api.NewServer(discoveryEngine, metricsClient, limitsAnalyzer, driftDetector, capacityPlanner, llmAssistant)

	// Setup Gin router
	router := gin.Default()
	
	// Add middleware
	router.Use(gin.Recovery())
	router.Use(api.CORSMiddleware())
	router.Use(api.LoggingMiddleware())
	
	// API routes
	apiGroup := router.Group("/api")
		apiGroup.GET("/health", server.HealthCheck)
		apiGroup.GET("/tenants", server.GetTenants)
		apiGroup.GET("/tenants/:name", server.GetTenant)
		apiGroup.PUT("/tenants/:name/alloy/replicas", server.UpdateAlloyReplicas)
		apiGroup.GET("/limits", server.GetLimits)
		apiGroup.GET("/limits/:tenant", server.GetTenantLimits)
		apiGroup.POST("/limits/:tenant/apply", server.ApplyLimits)
		apiGroup.GET("/config", server.GetConfig)
		apiGroup.GET("/audit", server.GetAuditLogs)
		apiGroup.POST("/analyze", server.AnalyzeTenant)
		apiGroup.GET("/drift", server.GetDriftStatus)
		apiGroup.POST("/drift/:tenant/fix", server.FixDrift)
		apiGroup.GET("/capacity", server.GetCapacityReport)
		apiGroup.POST("/capacity/:tenant/generate", server.GenerateCapacityReport)
		apiGroup.GET("/llm/query", server.QueryLLM)
		apiGroup.POST("/export/:format", server.ExportData)
	}

	// Metrics endpoint (separate from API group)
	router.GET("/metrics", server.GetMetrics)

	// Serve static files for UI
	router.Static("/dashboard", "./web-ui/build")
	router.StaticFile("/", "./web-ui/build/index.html")

	// Get port from config
	port := viper.GetString("server.port")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logrus.Infof("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Start background services
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Periodic drift detection
				if err := driftDetector.DetectDrift(context.Background()); err != nil {
					logrus.Warnf("Drift detection failed: %v", err)
				}
			}
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Fatalf("Server forced to shutdown: %v", err)
	}

	logrus.Info("Server exited")
} 