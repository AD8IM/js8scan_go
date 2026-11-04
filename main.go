package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

func main() {
	cfgPath := flag.String("config", "config.ini", "path to INI config file")
	flag.Parse()

	cfg, err := LoadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	cl, err := Connect(cfg.Host, cfg.Port)
	if err != nil {
		log.Fatalf("connect to JS8Call API (%s:%d): %v", cfg.Host, cfg.Port, err)
	}
	defer cl.Close()
	log.Printf("connected to %s API at %s:%d", cfg.RadioName, cfg.Host, cfg.Port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	lines := make(chan LogLine, 256)
	if cfg.LogDir != "" {
		go TailFile(ctx, filepath.Join(cfg.LogDir, cfg.AllFile), "ALL", lines)
		go TailFile(ctx, filepath.Join(cfg.LogDir, cfg.DirectedFile), "DIRECTED", lines)
	}

	sc := NewScanner(cfg, cl)
	go sc.HandleEvents(ctx)
	go sc.HandleLogs(ctx, lines)

	cl.GetFreq()
	sc.Run(ctx)
	log.Println("shutting down")
}
