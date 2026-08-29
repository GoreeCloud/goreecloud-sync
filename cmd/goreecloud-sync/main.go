package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/GoreeCloud/goreecloud-sync/internal/app"
	"github.com/GoreeCloud/goreecloud-sync/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "serve":
		return runServe(args[1:])
	case "version", "--version", "-version":
		fmt.Printf("goreecloud-sync %s (%s)\n", version.Version, version.Commit)
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("listen", "127.0.0.1:8787", "HTTP listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return app.NewServer(*addr, logger).Run(ctx)
}

func printUsage() {
	fmt.Println(`GoreeCloud Sync

Usage:
  goreecloud-sync serve [--listen 127.0.0.1:8787]
  goreecloud-sync version
  goreecloud-sync help

The default CLI service exposes the base development shell. Authenticated first-party Sync ingestion and retrieval foundations exist in source but are not enabled by the default serve command. Nearby, Share, folder synchronization, production deployment, and Stable qualification remain incomplete.`)
}
