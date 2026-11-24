package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/Frobama/beacon-alerts/internal/domain"
	"github.com/Frobama/beacon-alerts/internal/store"
)

type Config struct {
	WebhookToken string
}

type Store interface {
	Record(ctx context.Context, a domain.Alert) (store.RecordResult, error)
	SetIncident(ctx context.Context, alertID, incidentID string) error
	Ping(ctx context.Context) error
}

type Reporter interface {
	ReportAlert(ctx context.Context, a domain.Alert) (incidentID string, created bool, err error)
}

type Server struct {
	cfg      Config
	store    Store
	reporter Reporter
	log      *slog.Logger
	resueltas atomic.Int64
}

func New(cfg Config, st Store, rep Reporter, log *slog.Logger) *Server {
	return &Server{cfg: cfg, store: st, reporter: rep, log: log}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhooks/alertmanager", s.handleWebhook)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /livez", s.handleLivez)
	return mux
}
