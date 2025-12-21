package main

// @title Go WhatsApp Multi-Device and Multi Session REST API
// @version 1.2.1
// @description Enterprise-grade WhatsApp Multi-Device REST API with 111+ endpoints including messaging, polls, newsletters/channels, status/stories, and 31 webhook event types

// @contact.name gdbrns
// @contact.url https://github.com/gdbrns/go-whatsapp-multi-session-rest-api

// @license.name MIT
// @license.url https://github.com/gdbrns/go-whatsapp-multi-session-rest-api/blob/main/LICENSE

// @host localhost:7001
// @BasePath /

// @securityDefinitions.apikey AdminAuth
// @in header
// @name X-Admin-Secret
// @description Admin secret key for managing API keys

// @securityDefinitions.apikey APIKeyAuth
// @in header
// @name X-API-Key
// @description API key for creating devices

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Bearer token for device operations

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	cron "github.com/robfig/cron/v3"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/env"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/router"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/internal"
)

type Server struct {
	Address string
	Port    string
}

func main() {
	var err error

	// Intialize Cron
	c := cron.New(cron.WithChain(
		cron.Recover(cron.DiscardLogger),
	), cron.WithSeconds())

	// Initialize Fiber
	app := fiber.New(fiber.Config{
		ErrorHandler:   router.HttpErrorHandler,
		BodyLimit:      router.BodyLimitBytes(),
		ReadBufferSize: 8192, // Increase from default 4096 to handle larger headers (JWT tokens)
	})

	// Request ID + panic recovery (structured JSON)
	app.Use(router.HttpRequestID())
	app.Use(router.RecoveryMiddleware())

	// Router Compression
	app.Use(compress.New(compress.Config{
		Level: compress.Level(router.GZipLevel),
		Next: func(c *fiber.Ctx) bool {
			return strings.Contains(c.Path(), "docs")
		},
	}))

	// Router CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: router.CORSOrigin,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE",
	}))

	// Router Security
	app.Use(helmet.New(helmet.Config{
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "SAMEORIGIN",
	}))

	// Router Cache
	app.Use(router.HttpCacheInMemory(
		router.CacheCapacity,
		router.CacheTTLSeconds,
	))

	// Router RealIP + request context enrichment
	app.Use(router.HttpRealIP())

	// Router Default Handler
	app.Get("/favicon.ico", router.ResponseNoContent)

	// Load Internal Routes
	internal.Routes(app)

	// Running Startup Tasks
	internal.Startup()

	// Running Routines Tasks
	internal.Routines(c)

	// Get Server Configuration with defaults
	var serverConfig Server

	// SERVER_ADDRESS: default "0.0.0.0" (all interfaces)
	serverConfig.Address = env.GetEnvStringOrDefault("SERVER_ADDRESS", "0.0.0.0")

	// SERVER_PORT: default "7001"
	serverConfig.Port = env.GetEnvStringOrDefault("SERVER_PORT", "7001")

	// Start Server
	go func() {
		if err := app.Listen(serverConfig.Address + ":" + serverConfig.Port); err != nil {
			log.Print(nil).Fatal(err.Error())
		}
	}()

	// Watch for Shutdown Signal
	sigShutdown := make(chan os.Signal, 1)
	signal.Notify(sigShutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-sigShutdown
	// Wait 5 Seconds Before Graceful Shutdown
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	// Try To Shutdown Server
	err = app.ShutdownWithContext(ctxShutdown)
	if err != nil {
		log.Print(nil).Fatal(err.Error())
	}

	// Try To Shutdown Cron
	c.Stop()
}
