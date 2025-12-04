package main

// @title Go WhatsApp Multi-Device and Multi Session REST API
// @version 1.0.0
// @description This is WhatsApp Multi-Device and Multi Session Implementation in Go REST API

// @contact.name Dimas Restu Hidayanto

// @securityDefinitions.basic BasicAuth

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

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
	"github.com/gofiber/fiber/v2/middleware/recover"

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

	// Router Recovery
	app.Use(recover.New())

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

	// Router RealIP
	app.Use(router.HttpRealIP())

	// Router Default Handler
	app.Get("/favicon.ico", router.ResponseNoContent)

	// Load Internal Routes
	internal.Routes(app)

	// Running Startup Tasks
	internal.Startup()

	// Running Routines Tasks
	internal.Routines(c)

	// Get Server Configuration
	var serverConfig Server

	serverConfig.Address, err = env.GetEnvString("SERVER_ADDRESS")
	if err != nil {
		serverConfig.Address = "127.0.0.1"
	}

	serverConfig.Port, err = env.GetEnvString("SERVER_PORT")
	if err != nil {
		serverConfig.Port = "8000"
	}

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
