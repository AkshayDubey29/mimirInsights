package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akshaydubey29/mimirInsights/pkg/api"
	"github.com/akshaydubey29/mimirInsights/pkg/config"
	"github.com/akshaydubey29/mimirInsights/pkg/discovery"
	"github.com/akshaydubey29/mimirInsights/pkg/limits"
	"github.com/akshaydubey29/mimirInsights/pkg/metrics"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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

	// Initialize components
	discoveryEngine := discovery.NewEngine()
	metricsClient := metrics.NewClient()
	limitsAnalyzer := limits.NewAnalyzer(metricsClient)

	// Create API server
	server := api.NewServer(discoveryEngine, metricsClient, limitsAnalyzer)

	// Setup Gin router
	router := gin.Default()

	// Add CORS middleware
	router.Use(gin.Recovery())
	router.Use(api.CORSMiddleware())

	// Health check endpoints for Kubernetes
	router.GET("/ready", server.HealthCheck)
	router.GET("/healthz", server.HealthCheck)

	// API routes
	apiGroup := router.Group("/api")
	{
		apiGroup.GET("/health", server.HealthCheck)
		apiGroup.GET("/tenants", server.GetTenants)
		apiGroup.POST("/tenants", server.CreateTenant)
		apiGroup.GET("/limits", server.GetLimits)
		apiGroup.GET("/config", server.GetConfig)
		apiGroup.GET("/environment", server.GetEnvironment)
		apiGroup.GET("/metrics", server.GetMetrics)
		apiGroup.GET("/metrics/discovery", server.GetAutoDiscoveredMetrics)
		apiGroup.GET("/audit", server.GetAuditLogs)
		apiGroup.POST("/analyze", server.AnalyzeTenant)
		apiGroup.GET("/drift", server.GetDriftStatus)
		apiGroup.POST("/drift/baseline", server.CreateDriftBaseline)
		apiGroup.GET("/alloy/deployments", server.GetAlloyDeployments)
		apiGroup.GET("/alloy/workloads", server.GetAlloyWorkloads)
		apiGroup.POST("/alloy/scale", server.ScaleAlloyReplicas)
		apiGroup.GET("/alloy/recommendations", server.GetAlloyScalingRecommendations)
		apiGroup.GET("/capacity", server.GetCapacityReport)
		apiGroup.GET("/capacity/export", server.ExportCapacityReport)
		apiGroup.GET("/capacity/trends", server.GetCapacityTrends)
		apiGroup.POST("/llm/query", server.ProcessLLMQuery)
		apiGroup.GET("/llm/capabilities", server.GetLLMCapabilities)
	}

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
	}

	// Start server in a goroutine
	go func() {
		logrus.Infof("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start server: %v", err)
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
