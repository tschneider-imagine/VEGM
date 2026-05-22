package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/tschneider-imagine/VEGM/runtime"
)

func main() {
	configPath := flag.String("config", "", "path to VEGM runtime config JSON")
	flag.Parse()
	if *configPath == "" {
		log.Fatal("-config is required")
	}
	cfg, err := runtime.LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	srv, err := runtime.NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := srv.Start(ctx); err != nil {
		log.Fatal(err)
	}
	srv.StartSessionEngine(ctx)
	fmt.Printf("VEGM wire=%s control=%s\n", srv.WireAddr(), srv.ControlAddr())
	<-ctx.Done()
	_ = srv.Shutdown(context.Background())
}
