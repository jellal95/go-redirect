package utils

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
)

// GracefulShutdown handles graceful shutdown for Fiber app
func GracefulShutdown(app *fiber.App, timeout time.Duration) {
	// Create channel to listen for interrupt signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block until signal is received
	sig := <-c
	fmt.Printf("\n🛑 Received signal: %v. Shutting down gracefully...\n", sig)

	LogInfo(LogEntry{
		Type:  "server_shutdown",
		Extra: map[string]interface{}{"signal": sig.String()},
	})

	// Create context with timeout
	if err := app.ShutdownWithTimeout(timeout); err != nil {
		LogFatal(LogEntry{
			Type:  "shutdown_error",
			Extra: map[string]interface{}{"error": err.Error()},
		}, 1)
	}

	fmt.Println("✅ Server shutdown complete")
}
