package grpcapi

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	alertsv1 "github.com/Frobama/beacon-alerts/gen/beacon/alerts/v1"
	"github.com/Frobama/beacon-alerts/internal/domain"
)

func toProto(a domain.Alert) *alertsv1.Alert {
	return &alertsv1.Alert{
		Id:              a.ID,
		Source:          a.Source,
		Fingerprint:     a.Fingerprint,
		Title:           a.Title,
		Service:         a.Service,
		Severity:        a.Severity,
		Labels:          a.Labels,
		RawPayload:      string(a.RawPayload),
		Occurrences:     a.Occurrences,
		IncidentId:      a.IncidentID,
		FirstReceivedAt: timestamppb.New(a.FirstReceivedAt),
		LastReceivedAt:  timestamppb.New(a.LastReceivedAt),
	}
}
