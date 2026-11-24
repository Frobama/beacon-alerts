package incidentsclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	incidentsv1 "github.com/Frobama/beacon-alerts/gen/beacon/incidents/v1"
	"github.com/Frobama/beacon-alerts/internal/domain"
)

type Client struct {
	conn    *grpc.ClientConn
	rpc     incidentsv1.IncidentServiceClient
	timeout time.Duration
}

func Dial(target string, timeout time.Duration) (*Client, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("conectar a incidentes: %w", err)
	}
	return New(conn, timeout), nil
}

func New(conn *grpc.ClientConn, timeout time.Duration) *Client {
	return &Client{conn: conn, rpc: incidentsv1.NewIncidentServiceClient(conn), timeout: timeout}
}

// ReportAlert pide a Incidentes que abra un incidente o asocie la alerta.
// Devuelve el incidente y si fue nuevo.
func (c *Client) ReportAlert(ctx context.Context, a domain.Alert) (string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.rpc.ReportAlert(ctx, &incidentsv1.ReportAlertRequest{
		AlertId:     a.ID,
		Fingerprint: a.Fingerprint,
		Source:      a.Source,
		Title:       a.Title,
		Service:     a.Service,
		Severity:    a.Severity,
		Labels:      a.Labels,
		ReceivedAt:  timestamppb.New(a.LastReceivedAt),
	})
	if err != nil {
		return "", false, err
	}
	return resp.GetIncidentId(), resp.GetIncidentCreated(), nil
}

func (c *Client) Close() error { return c.conn.Close() }
