package main

import (
	"context"
	"etlgo/internal/delivery"
	"etlgo/internal/infrastructure"
	"etlgo/internal/usecase"
	"etlgo/pkg/config"
	"etlgo/pkg/logger"
	"etlgo/pkg/metrics"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Logging.Level)
	log.Info("Starting server")

	metrics := metrics.New()

	// Initialize repositories
	adRepo := infrastructure.NewAdRepository(log)
	crmRepo := infrastructure.NewCRMRepository(log)
	metricsRepo := infrastructure.NewMetricsRepository(log)

	// Initialize HTTP client
	httpClient := infrastructure.NewHTTPClient(
		cfg.External.AdsAPIURL,
		cfg.External.CRMAPIURL,
		cfg.External.SinkURL,
		cfg.External.SinkSecret,
		cfg.ETL.RequestTimeout,
		log,
		metrics,
	)

	// Initialize services
	etlService := usecase.NewETLService(
		adRepo,
		crmRepo,
		metricsRepo,
		httpClient,
		log,
		metrics,
		cfg.ETL.WorkerPoolSize,
		cfg.ETL.BatchSize,
	)

	metricsService := usecase.NewMetricsService(
		metricsRepo,
		httpClient,
		log,
		metrics,
	)

	handlers := delivery.NewHTTPHandlers(
		etlService,
		metricsService,
		log,
		metrics,
	)

	// Initialize router
	router := delivery.NewHTTPRouter(handlers, log, metrics)
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router.SetupRoutes(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// Start the server
	go func() {
		log.WithField("port", cfg.Server.Port).Info("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.WithError(err).Error("Server forced to shutdown")
		os.Exit(1)
	}

	log.Info("Server exited")
}
