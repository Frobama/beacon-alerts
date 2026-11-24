package grpcapi

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	alertsv1 "github.com/Frobama/beacon-alerts/gen/beacon/alerts/v1"
	"github.com/Frobama/beacon-alerts/internal/domain"
	"github.com/Frobama/beacon-alerts/internal/store"
)

type Store interface {
	Get(ctx context.Context, id string) (domain.Alert, error)
	List(ctx context.Context, f store.ListFilter) (store.ListResult, error)
	ListByIncident(ctx context.Context, incidentID string) ([]domain.Alert, error)
}

type Server struct {
	alertsv1.UnimplementedAlertServiceServer
	store Store
}

func New(st Store) *Server { return &Server{store: st} }

func (s *Server) GetAlert(ctx context.Context, req *alertsv1.GetAlertRequest) (*alertsv1.GetAlertResponse, error) {
	if _, err := uuid.Parse(req.GetId()); err != nil {
		return nil, status.Error(codes.InvalidArgument, "id de alerta inválido")
	}
	a, err := s.store.Get(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "alerta no encontrada")
		}
		return nil, status.Error(codes.Internal, "no se pudo leer la alerta")
	}
	return &alertsv1.GetAlertResponse{Alert: toProto(a)}, nil
}

func (s *Server) ListAlerts(ctx context.Context, req *alertsv1.ListAlertsRequest) (*alertsv1.ListAlertsResponse, error) {
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	if req.GetPageToken() != "" {
		if _, _, err := store.DecodeToken(req.GetPageToken()); err != nil {
			return nil, status.Error(codes.InvalidArgument, "token de página inválido")
		}
	}
	res, err := s.store.List(ctx, store.ListFilter{
		Source:    req.GetSource(),
		Severity:  req.GetSeverity(),
		Service:   req.GetService(),
		PageSize:  pageSize,
		PageToken: req.GetPageToken(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "filtros inválidos")
	}
	resp := &alertsv1.ListAlertsResponse{NextPageToken: res.NextPageToken}
	for _, a := range res.Alerts {
		resp.Alerts = append(resp.Alerts, toProto(a))
	}
	return resp, nil
}

func (s *Server) GetAlertsByIncident(ctx context.Context, req *alertsv1.GetAlertsByIncidentRequest) (*alertsv1.GetAlertsByIncidentResponse, error) {
	if _, err := uuid.Parse(req.GetIncidentId()); err != nil {
		return nil, status.Error(codes.InvalidArgument, "id de incidente inválido")
	}
	alertas, err := s.store.ListByIncident(ctx, req.GetIncidentId())
	if err != nil {
		return nil, status.Error(codes.Internal, "no se pudieron leer las alertas")
	}
	resp := &alertsv1.GetAlertsByIncidentResponse{}
	for _, a := range alertas {
		resp.Alerts = append(resp.Alerts, toProto(a))
	}
	return resp, nil
}

func NewGRPCServer(st Store) *grpc.Server {
	g := grpc.NewServer()
	alertsv1.RegisterAlertServiceServer(g, New(st))
	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(g, healthSrv)
	reflection.Register(g)
	return g
}
