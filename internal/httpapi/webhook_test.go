package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Frobama/beacon-alerts/internal/domain"
	"github.com/Frobama/beacon-alerts/internal/store"
)

type storeFalso struct {
	creadas   int32
	falla     error
	asociados []string
}

func (s *storeFalso) Record(_ context.Context, a domain.Alert) (store.RecordResult, error) {
	if s.falla != nil {
		return store.RecordResult{}, s.falla
	}
	atomic.AddInt32(&s.creadas, 1)
	a.ID = "a-1"
	return store.RecordResult{Alert: a, Created: true}, nil
}
func (s *storeFalso) SetIncident(_ context.Context, alertID, incidentID string) error {
	s.asociados = append(s.asociados, alertID+"="+incidentID)
	return nil
}
func (s *storeFalso) Ping(context.Context) error { return nil }

type reporterFalso struct {
	llamadas int32
	err      error
}

func (r *reporterFalso) ReportAlert(context.Context, domain.Alert) (string, bool, error) {
	atomic.AddInt32(&r.llamadas, 1)
	if r.err != nil {
		return "", false, r.err
	}
	return "inc-1", true, nil
}

func servidorDePrueba(st Store, rep Reporter, token string) http.Handler {
	return New(Config{WebhookToken: token}, st, rep, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()
}

const payload = `{"version":"4","status":"firing","alerts":[{"status":"firing","labels":{"alertname":"Latencia","service":"checkout","severity":"critical"},"annotations":{"summary":"Latencia alta"}}]}`

func TestWebhookSinTokenRechaza(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/alertmanager", strings.NewReader(payload))
	servidorDePrueba(&storeFalso{}, &reporterFalso{}, "secreto").ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("código = %d", rec.Code)
	}
}

func TestWebhookTokenIncorrectoRechaza(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/alertmanager", strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer otro")
	servidorDePrueba(&storeFalso{}, &reporterFalso{}, "secreto").ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("código = %d", rec.Code)
	}
}

func TestWebhookCuerpoInvalido(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/alertmanager", strings.NewReader("{no es json"))
	servidorDePrueba(&storeFalso{}, &reporterFalso{}, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("código = %d", rec.Code)
	}
}

func TestWebhookCuerpoDemasiadoGrande(t *testing.T) {
	grande := `{"alerts":[{"annotations":{"summary":"` + strings.Repeat("x", 2<<20) + `"}}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/alertmanager", strings.NewReader(grande))
	servidorDePrueba(&storeFalso{}, &reporterFalso{}, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("código = %d", rec.Code)
	}
}

func TestWebhookAceptaGuardaYReporta(t *testing.T) {
	st := &storeFalso{}
	rep := &reporterFalso{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/alertmanager", strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer secreto")
	servidorDePrueba(st, rep, "secreto").ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("código = %d, cuerpo %s", rec.Code, rec.Body.String())
	}
	if st.creadas != 1 || rep.llamadas != 1 {
		t.Fatalf("creadas = %d, reportes = %d", st.creadas, rep.llamadas)
	}
	if len(st.asociados) != 1 || st.asociados[0] != "a-1=inc-1" {
		t.Fatalf("asociados = %v", st.asociados)
	}
}

func TestWebhookIgnoraResueltas(t *testing.T) {
	st := &storeFalso{}
	rep := &reporterFalso{}
	resuelta := `{"status":"resolved","alerts":[{"status":"resolved","labels":{"alertname":"X"}}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/alertmanager", strings.NewReader(resuelta))
	servidorDePrueba(st, rep, "").ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted || st.creadas != 0 || rep.llamadas != 0 {
		t.Fatalf("código = %d, creadas = %d, reportes = %d", rec.Code, st.creadas, rep.llamadas)
	}
}

func TestWebhookSigueSiIncidentesFalla(t *testing.T) {
	st := &storeFalso{}
	rep := &reporterFalso{err: errors.New("caído")}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/alertmanager", strings.NewReader(payload))
	servidorDePrueba(st, rep, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("código = %d", rec.Code)
	}
	if len(st.asociados) != 0 {
		t.Fatalf("no debía asociar: %v", st.asociados)
	}
}

func TestWebhookFallaElGuardadoDevuelve500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/alertmanager", strings.NewReader(payload))
	servidorDePrueba(&storeFalso{falla: errors.New("db caída")}, &reporterFalso{}, "").ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("código = %d", rec.Code)
	}
}
