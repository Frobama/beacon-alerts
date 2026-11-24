package grpcapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	alertsv1 "github.com/Frobama/beacon-alerts/gen/beacon/alerts/v1"
	commonv1 "github.com/Frobama/beacon-alerts/gen/beacon/common/v1"
	"github.com/Frobama/beacon-alerts/internal/domain"
	"github.com/Frobama/beacon-alerts/internal/store"
)

type storeFalso struct {
	alertas map[string]domain.Alert
	filtro  store.ListFilter
}

func (s *storeFalso) Get(_ context.Context, id string) (domain.Alert, error) {
	a, ok := s.alertas[id]
	if !ok {
		return domain.Alert{}, store.ErrNotFound
	}
	return a, nil
}
func (s *storeFalso) List(_ context.Context, f store.ListFilter) (store.ListResult, error) {
	s.filtro = f
	return store.ListResult{Alerts: []domain.Alert{s.alertas["a-1"]}}, nil
}
func (s *storeFalso) ListByIncident(context.Context, string) ([]domain.Alert, error) {
	return []domain.Alert{s.alertas["a-1"]}, nil
}

const id = "11111111-1111-1111-1111-111111111111"

func servidorDePrueba() *Server {
	return New(&storeFalso{alertas: map[string]domain.Alert{id: {
		ID: id, Source: "alertmanager", Fingerprint: "fp", Title: "t", Service: "checkout",
		Severity: commonv1.Severity_SEVERITY_HIGH, Occurrences: 3,
		FirstReceivedAt: time.Now().UTC(), LastReceivedAt: time.Now().UTC(),
	}}})
}

func TestGetAlertUUIDInvalido(t *testing.T) {
	_, err := servidorDePrueba().GetAlert(context.Background(), &alertsv1.GetAlertRequest{Id: "no-es-uuid"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestGetAlertNoExiste(t *testing.T) {
	_, err := servidorDePrueba().GetAlert(context.Background(), &alertsv1.GetAlertRequest{Id: "22222222-2222-2222-2222-222222222222"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("err = %v", err)
	}
}

func TestGetAlertMapea(t *testing.T) {
	resp, err := servidorDePrueba().GetAlert(context.Background(), &alertsv1.GetAlertRequest{Id: id})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Alert.Id != id || resp.Alert.Occurrences != 3 || resp.Alert.Severity != commonv1.Severity_SEVERITY_HIGH {
		t.Fatalf("alerta = %+v", resp.Alert)
	}
}

func TestListAlertsRecortaPageSize(t *testing.T) {
	st := &storeFalso{alertas: map[string]domain.Alert{id: {ID: id}}}
	s := New(st)
	if _, err := s.ListAlerts(context.Background(), &alertsv1.ListAlertsRequest{PageSize: 0}); err != nil {
		t.Fatal(err)
	}
	if st.filtro.PageSize != 20 {
		t.Fatalf("page_size 0 → %d", st.filtro.PageSize)
	}
	if _, err := s.ListAlerts(context.Background(), &alertsv1.ListAlertsRequest{PageSize: 500}); err != nil {
		t.Fatal(err)
	}
	if st.filtro.PageSize != 100 {
		t.Fatalf("page_size 500 → %d", st.filtro.PageSize)
	}
}

func TestListAlertsTokenInvalido(t *testing.T) {
	_, err := servidorDePrueba().ListAlerts(context.Background(), &alertsv1.ListAlertsRequest{PageToken: "%%%"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

func TestGetAlertsByIncidentValidaUUID(t *testing.T) {
	_, err := servidorDePrueba().GetAlertsByIncident(context.Background(), &alertsv1.GetAlertsByIncidentRequest{IncidentId: "x"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v", err)
	}
}

var _ = errors.Is
