package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("INCIDENTS_GRPC_ADDR", "localhost:9090")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("GRPC_ADDR", "")
	t.Setenv("DEDUP_WINDOW", "")
	t.Setenv("REPORT_TIMEOUT", "")
	t.Setenv("RECONCILE_INTERVAL", "")
	t.Setenv("RECONCILE_MARGIN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8082" || cfg.GRPCAddr != ":9091" {
		t.Fatalf("direcciones = %q/%q", cfg.HTTPAddr, cfg.GRPCAddr)
	}
	if cfg.DedupWindow != 30*time.Minute || cfg.ReportTimeout != 5*time.Second {
		t.Fatalf("duraciones = %v/%v", cfg.DedupWindow, cfg.ReportTimeout)
	}
	if cfg.ReconcileInterval != 30*time.Second || cfg.ReconcileMargin != 10*time.Second {
		t.Fatalf("reconciliación = %v/%v", cfg.ReconcileInterval, cfg.ReconcileMargin)
	}
}

func TestLoadRequeridos(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("INCIDENTS_GRPC_ADDR", "")
	if _, err := Load(); err == nil {
		t.Fatal("quería error por variables requeridas")
	}
}

func TestLoadDuracionesInvalidas(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("INCIDENTS_GRPC_ADDR", "localhost:9090")
	t.Setenv("DEDUP_WINDOW", "no-es-duracion")
	if _, err := Load(); err == nil {
		t.Fatal("quería error por duración inválida")
	}
}

func TestLoadMargenMayorQueIntervalo(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("INCIDENTS_GRPC_ADDR", "localhost:9090")
	t.Setenv("RECONCILE_INTERVAL", "10s")
	t.Setenv("RECONCILE_MARGIN", "20s")
	if _, err := Load(); err == nil {
		t.Fatal("quería error: el margen no puede superar el intervalo")
	}
}

func TestMigrateOnStart(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("INCIDENTS_GRPC_ADDR", "localhost:9090")

	casos := []struct {
		env     string
		want    bool
		wantErr bool
	}{
		{"", true, false},
		{"true", true, false},
		{"false", false, false},
		{"0", false, false},
		{"quizá", false, true},
	}

	for _, c := range casos {
		t.Setenv("MIGRATE_ON_START", c.env)
		cfg, err := Load()
		if c.wantErr {
			if err == nil {
				t.Fatalf("%q: quería error", c.env)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: error inesperado: %v", c.env, err)
		}
		if cfg.MigrateOnStart != c.want {
			t.Fatalf("%q: MigrateOnStart = %v, quería %v", c.env, cfg.MigrateOnStart, c.want)
		}
	}
}
