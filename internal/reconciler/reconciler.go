package reconciler

import (
	"context"
	"log/slog"
	"time"

	"github.com/Frobama/beacon-alerts/internal/domain"
)

type Store interface {
	ListUnassigned(ctx context.Context, olderThan time.Time, limit int) ([]domain.Alert, error)
	SetIncident(ctx context.Context, alertID, incidentID string) error
}

type Reporter interface {
	ReportAlert(ctx context.Context, a domain.Alert) (incidentID string, created bool, err error)
}

type Config struct {
	Interval time.Duration
	Margin   time.Duration
}

type Reconciler struct {
	store    Store
	reporter Reporter
	cfg      Config
	log      *slog.Logger
}

func New(st Store, rep Reporter, cfg Config, log *slog.Logger) *Reconciler {
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Second
	}
	return &Reconciler{store: st, reporter: rep, cfg: cfg, log: log}
}

// Run reconcilia cada intervalo hasta que se cancele el contexto.
func (r *Reconciler) Run(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.Once(ctx); err != nil {
				r.log.Error("reconciliación", "err", err)
			}
		}
	}
}

// Once reintenta las alertas guardadas sin incidente.
func (r *Reconciler) Once(ctx context.Context) error {
	corte := time.Now().UTC().Add(-r.cfg.Margin)
	pendientes, err := r.store.ListUnassigned(ctx, corte, 100)
	if err != nil {
		return err
	}
	for _, a := range pendientes {
		incidentID, _, err := r.reporter.ReportAlert(ctx, a)
		if err != nil {
			r.log.Warn("reconciliar alerta", "alert_id", a.ID, "err", err)
			continue
		}
		if err := r.store.SetIncident(ctx, a.ID, incidentID); err != nil {
			r.log.Warn("asociar alerta reconciliada", "alert_id", a.ID, "err", err)
			continue
		}
		r.log.Info("alerta reconciliada", "alert_id", a.ID, "incident_id", incidentID)
	}
	return nil
}
