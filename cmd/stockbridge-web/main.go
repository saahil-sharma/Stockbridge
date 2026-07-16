package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"stockbridge/internal/app"
	"stockbridge/internal/web"
)

func main() {
	if err := run(); err != nil {
		log.Printf("stockbridge web server stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	addrFlag := flag.String("addr", "", "HTTP server address (default :$PORT or :8080)")
	flag.Parse()

	addr, err := listenAddress(*addrFlag, os.Getenv("PORT"))
	if err != nil {
		return err
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	if strings.TrimSpace(os.Getenv("STOCKBRIDGE_SEC_USER_AGENT")) == "" {
		logger.Print("warning: STOCKBRIDGE_SEC_USER_AGENT is not set; configure it before public deployment")
	}

	httpClient := &http.Client{Timeout: 20 * time.Second}
	server := web.NewServer(app.NewAnalyzer(httpClient))
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          logger,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()

	logger.Printf("Stockbridge web app listening at %s", displayURL(addr))
	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdownSignal.Done():
	}

	logger.Print("shutting down Stockbridge web app")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func listenAddress(explicitAddress, providerPort string) (string, error) {
	if address := strings.TrimSpace(explicitAddress); address != "" {
		return address, nil
	}

	port := strings.TrimSpace(providerPort)
	if port == "" {
		port = "8080"
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", fmt.Errorf("invalid PORT %q: expected a number from 1 to 65535", providerPort)
	}
	return ":" + port, nil
}

func displayURL(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	return "http://" + addr
}
