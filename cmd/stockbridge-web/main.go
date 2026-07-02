package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"stockbridge/internal/app"
	"stockbridge/internal/web"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP server address")
	flag.Parse()

	httpClient := &http.Client{Timeout: 20 * time.Second}
	server := web.NewServer(app.NewAnalyzer(httpClient))

	fmt.Printf("Stockbridge web app running at http://%s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, server.Routes()))
}
