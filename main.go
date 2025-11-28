package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/notblessy/dexlite/db"
	"github.com/notblessy/dexlite/handlers"
	"github.com/notblessy/dexlite/models"
	"github.com/notblessy/dexlite/workers"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
}

func main() {
	// Initialize database
	database := db.NewPostgres()

	// Auto-migrate the schema
	if err := database.AutoMigrate(&models.CoinPrice{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database initialized and migrated successfully")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create workers
	priceFetcher := workers.NewPriceFetcher(database)
	cleanupWorker := workers.NewCleanupWorker(database)

	// Fetch initial prices synchronously before starting background workers
	log.Println("Fetching initial coin prices...")
	priceFetcher.FetchPrices()

	// WaitGroup to wait for all workers to finish
	var wg sync.WaitGroup

	// Start workers in separate goroutines
	wg.Add(2)
	go func() {
		defer wg.Done()
		priceFetcher.Start(ctx)
	}()
	go func() {
		defer wg.Done()
		cleanupWorker.Start(ctx)
	}()

	log.Println("Workers started successfully")
	log.Println("Price fetcher running every hour")
	log.Println("Cleanup worker running every hour")

	// Setup HTTP server with Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	// Initialize handlers
	priceHandler := handlers.NewPriceHandler(database)

	// Setup routes
	api := e.Group("/api")
	api.GET("/prices/:coin", priceHandler.GetPriceComparison)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start HTTP server in a goroutine
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: e,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("HTTP server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutdown signal received, initiating graceful shutdown...")

	// Cancel context to signal workers to stop
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Println("HTTP server stopped successfully")
	}

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers stopped successfully")
	case <-time.After(30 * time.Second):
		log.Println("Timeout waiting for workers to stop, forcing shutdown")
	}

	log.Println("Application shutdown complete")
}
