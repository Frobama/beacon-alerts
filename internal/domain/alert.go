package domain

import (
	"time"

	commonv1 "github.com/Frobama/beacon-alerts/gen/beacon/common/v1"
)

// Alert es una alerta ya normalizada y persistida.
type Alert struct {
	ID              string
	Source          string
	Fingerprint     string
	Title           string
	Service         string
	Severity        commonv1.Severity
	Labels          map[string]string
	RawPayload      []byte
	Occurrences     int32
	IncidentID      string
	FirstReceivedAt time.Time
	LastReceivedAt  time.Time
}
