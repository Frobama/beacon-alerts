package incidentsclient

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	commonv1 "github.com/Frobama/beacon-alerts/gen/beacon/common/v1"
	incidentsv1 "github.com/Frobama/beacon-alerts/gen/beacon/incidents/v1"
	"github.com/Frobama/beacon-alerts/internal/domain"
)

type servidorFalso struct {
	incidentsv1.UnimplementedIncidentServiceServer
	demora   time.Duration
	recibido *incidentsv1.ReportAlertRequest
}

func (s *servidorFalso) ReportAlert(ctx context.Context, req *incidentsv1.ReportAlertRequest) (*incidentsv1.ReportAlertResponse, error) {
	if s.demora > 0 {
		time.Sleep(s.demora)
	}
	s.recibido = req
	return &incidentsv1.ReportAlertResponse{IncidentId: "inc-1", IncidentCreated: true}, nil
}

func clienteDePrueba(t *testing.T, srv incidentsv1.IncidentServiceServer, timeout time.Duration) *Client {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	g := grpc.NewServer()
	incidentsv1.RegisterIncidentServiceServer(g, srv)
	go func() { _ = g.Serve(lis) }()
	t.Cleanup(g.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return New(conn, timeout)
}

func TestReportAlertMapeaLaAlerta(t *testing.T) {
	srv := &servidorFalso{}
	c := clienteDePrueba(t, srv, time.Second)

	incidentID, created, err := c.ReportAlert(context.Background(), domain.Alert{
		ID: "a-1", Fingerprint: "fp", Source: "alertmanager", Title: "t", Service: "checkout",
		Severity: commonv1.Severity_SEVERITY_CRITICAL, Labels: map[string]string{"k": "v"},
		LastReceivedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if incidentID != "inc-1" || !created {
		t.Fatalf("respuesta = %q/%v", incidentID, created)
	}
	if srv.recibido.AlertId != "a-1" || srv.recibido.Severity != commonv1.Severity_SEVERITY_CRITICAL {
		t.Fatalf("request = %+v", srv.recibido)
	}
}

func TestReportAlertRespetaElTimeout(t *testing.T) {
	c := clienteDePrueba(t, &servidorFalso{demora: 200 * time.Millisecond}, 20*time.Millisecond)
	_, _, err := c.ReportAlert(context.Background(), domain.Alert{ID: "a-1"})
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("err = %v, quería DeadlineExceeded", err)
	}
}

func TestReportAlertDevuelveErrorDelServidor(t *testing.T) {
	srv := &servidorFalsoError{}
	c := clienteDePrueba(t, srv, time.Second)
	_, _, err := c.ReportAlert(context.Background(), domain.Alert{ID: "a-1"})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("err = %v", err)
	}
}

type servidorFalsoError struct{ incidentsv1.UnimplementedIncidentServiceServer }

func (servidorFalsoError) ReportAlert(context.Context, *incidentsv1.ReportAlertRequest) (*incidentsv1.ReportAlertResponse, error) {
	return nil, status.Error(codes.Unavailable, "caído")
}

func TestDialYClose(t *testing.T) {
	c, err := Dial("127.0.0.1:1", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
