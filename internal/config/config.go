package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr          string
	GRPCAddr          string
	DatabaseURL       string
	IncidentsGRPCAddr string
	WebhookToken      string
	MigrateOnStart    bool
	DedupWindow       time.Duration
	ReportTimeout     time.Duration
	ReconcileInterval time.Duration
	ReconcileMargin   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:          envOr("HTTP_ADDR", ":8082"),
		GRPCAddr:          envOr("GRPC_ADDR", ":9091"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		IncidentsGRPCAddr: os.Getenv("INCIDENTS_GRPC_ADDR"),
		WebhookToken:      os.Getenv("WEBHOOK_TOKEN"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL es obligatoria")
	}
	if cfg.IncidentsGRPCAddr == "" {
		return Config{}, fmt.Errorf("INCIDENTS_GRPC_ADDR es obligatoria")
	}

	migrateOnStart := true
	if raw := os.Getenv("MIGRATE_ON_START"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("MIGRATE_ON_START inválida %q: %w", raw, err)
		}
		migrateOnStart = parsed
	}
	cfg.MigrateOnStart = migrateOnStart

	var err error
	if cfg.DedupWindow, err = duration("DEDUP_WINDOW", 30*time.Minute); err != nil {
		return Config{}, err
	}
	if cfg.ReportTimeout, err = duration("REPORT_TIMEOUT", 5*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.ReconcileInterval, err = duration("RECONCILE_INTERVAL", 30*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.ReconcileMargin, err = duration("RECONCILE_MARGIN", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.ReconcileMargin >= cfg.ReconcileInterval {
		return Config{}, fmt.Errorf("RECONCILE_MARGIN (%s) debe ser menor que RECONCILE_INTERVAL (%s)",
			cfg.ReconcileMargin, cfg.ReconcileInterval)
	}
	return cfg, nil
}

func LoadMigrate() (Config, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return Config{}, fmt.Errorf("DATABASE_URL es obligatoria")
	}
	return Config{DatabaseURL: url}, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func duration(key string, def time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s inválida %q: %w", key, raw, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s debe ser positiva, llegó %s", key, d)
	}
	return d, nil
}
