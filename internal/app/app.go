package app

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/Frobama/beacon-alerts/internal/config"
	"github.com/Frobama/beacon-alerts/internal/grpcapi"
	"github.com/Frobama/beacon-alerts/internal/httpapi"
	"github.com/Frobama/beacon-alerts/internal/incidentsclient"
	"github.com/Frobama/beacon-alerts/internal/reconciler"
	"github.com/Frobama/beacon-alerts/internal/store"
)

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	if cfg.MigrateOnStart {
		if err := store.Migrate(ctx, st.DB()); err != nil {
			return err
		}
		log.Info("migraciones aplicadas", "origen", "arranque")
	} else {
		log.Info("migraciones omitidas", "motivo", "MIGRATE_ON_START=false")
	}

	inc, err := incidentsclient.Dial(cfg.IncidentsGRPCAddr, cfg.ReportTimeout)
	if err != nil {
		return err
	}
	defer inc.Close()

	api := httpapi.New(httpapi.Config{WebhookToken: cfg.WebhookToken}, st, inc, log)
	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second}

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return err
	}
	grpcSrv := grpcapi.NewGRPCServer(st)

	rec := reconciler.New(st, inc, reconciler.Config{Interval: cfg.ReconcileInterval, Margin: cfg.ReconcileMargin}, log)

	errs := make(chan error, 2)
	go func() {
		log.Info("http escuchando", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- err
		}
	}()
	go func() {
		log.Info("grpc escuchando", "addr", cfg.GRPCAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			errs <- err
		}
	}()
	go rec.Run(ctx)

	select {
	case <-ctx.Done():
	case err := <-errs:
		log.Error("servidor caído", "err", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)

	terminado := make(chan struct{})
	go func() { grpcSrv.GracefulStop(); close(terminado) }()
	select {
	case <-terminado:
	case <-time.After(10 * time.Second):
		grpcSrv.Stop()
	}
	log.Info("apagado ordenado")
	return nil
}
