package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Frobama/beacon-alerts/internal/app"
	"github.com/Frobama/beacon-alerts/internal/config"
	"github.com/Frobama/beacon-alerts/internal/store"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	os.Exit(dispatch(os.Args[1:]))
}

func dispatch(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "version":
			fmt.Printf("beacon-alerts %s (%s)\n", version, commit)
			return 0
		case "migrate":
			if err := runMigrate(); err != nil {
				fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
				return 1
			}
			return 0
		default:
			fmt.Fprintf(os.Stderr, "uso: beacon-alerts [migrate|version]\n")
			return 2
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuración: %v\n", err)
		return 1
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, log); err != nil {
		log.Error("servicio", "err", err)
		return 1
	}
	return 0
}

func runMigrate() error {
	cfg, err := config.LoadMigrate()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	return store.Migrate(ctx, st.DB())
}
