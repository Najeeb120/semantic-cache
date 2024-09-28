package main

import (
	"context"
	"os"
	"os/signal"
	"semantic-cache/handlers"
	"syscall"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Message struct {
	Text string `json:"text"`
}

func main() {
	// configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Logger = log.With().Caller().Logger()

	app := fiber.New(fiber.Config{
		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,
	})

	// Provide a minimal config
	app.Use(healthcheck.New())

	// Check the cache to see if there is any data
	app.Get("/get", handlers.HandleGetRequest)

	// Put data in the cache
	app.Post("/post", handlers.HandlePutRequest)

	// Create a channel to listen for OS signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start the server in a goroutine
	go func() {
		log.Info().Msg("Server starting on :8080")
		if err := app.Listen(":8080"); err != nil && err != fiber.ErrInternalServerError {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Block until we receive a signal
	<-c

	log.Info().Msg("Gracefully shutting down...")

	// Create a deadline for the shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Attempt to gracefully shut down the server
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exiting")
}