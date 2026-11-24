package reconciler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Frobama/beacon-alerts/internal/domain"
)

type storeFalso struct {
	pendientes []domain.Alert
	asociados  map[string]string
}

func (s *storeFalso) ListUnassigned(context.Context, time.Time, int) ([]domain.Alert, error) {
	return s.pendientes, nil
}
func (s *storeFalso) SetIncident(_ context.Context, alertID, incidentID string) error {
	if s.asociados == nil {
		s.asociados = map[string]string{}
	}
	s.asociados[alertID] = incidentID
	return nil
}

type reporterFalso struct{ falla map[string]error }

func (r *reporterFalso) ReportAlert(_ context.Context, a domain.Alert) (string, bool, error) {
	if err := r.falla[a.ID]; err != nil {
		return "", false, err
	}
	return "inc-" + a.ID, true, nil
}

func TestOnceAsociaTodasLasPendientes(t *testing.T) {
	st := &storeFalso{pendientes: []domain.Alert{{ID: "a-1"}, {ID: "a-2"}}}
	r := New(st, &reporterFalso{}, Config{Margin: 10 * time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := r.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(st.asociados) != 2 || st.asociados["a-1"] != "inc-a-1" {
		t.Fatalf("asociados = %v", st.asociados)
	}
}

func TestOnceSigueSiUnaFalla(t *testing.T) {
	st := &storeFalso{pendientes: []domain.Alert{{ID: "a-1"}, {ID: "a-2"}}}
	rep := &reporterFalso{falla: map[string]error{"a-1": errors.New("caído")}}
	r := New(st, rep, Config{Margin: 10 * time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := r.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := st.asociados["a-1"]; ok {
		t.Fatal("no debía asociar la que falló")
	}
	if st.asociados["a-2"] != "inc-a-2" {
		t.Fatalf("asociados = %v", st.asociados)
	}
}

func TestRunTerminaConElContexto(t *testing.T) {
	r := New(&storeFalso{}, &reporterFalso{}, Config{Interval: 10 * time.Millisecond, Margin: time.Second},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { r.Run(ctx); close(done) }()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run no terminó")
	}
}
