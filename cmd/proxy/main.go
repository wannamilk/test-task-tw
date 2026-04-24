package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/wannamilk/test-task-tw/internal/proxy"
)

func main() {
	upstream := flag.String("upstream", envOrDefault("UPSTREAM_URL", "https://polygon.drpc.org"), "upstream RPC URL")
	addr := flag.String("addr", envOrDefault("LISTEN_ADDR", ":8080"), "listen address")
	flag.Parse()

	cfg := proxy.Config{
		UpstreamURL:    *upstream,
		ListenAddr:     *addr,
		RequestTimeout: 30 * time.Second,
		MaxBodyBytes:   1 << 20, // 1 MB
	}

	p, err := proxy.New(cfg)
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}

	log.Printf("starting proxy on %s → %s", cfg.ListenAddr, cfg.UpstreamURL)

	if err := http.ListenAndServe(cfg.ListenAddr, p); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}