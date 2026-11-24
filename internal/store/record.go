package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Frobama/beacon-alerts/internal/domain"
)

type RecordResult struct {
	Alert   domain.Alert
	Created bool
}

// Record guarda una recepción: si hay una alerta con la misma huella dentro de
// la ventana, suma la ocurrencia; si no, crea la alerta. Siempre deja el evento.
func (s *Store) Record(ctx context.Context, a domain.Alert) (RecordResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return RecordResult{}, err
	}
	defer tx.Rollback()

	// Serializa por huella: dos webhooks simultáneos no crean dos filas.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, a.Fingerprint); err != nil {
		return RecordResult{}, fmt.Errorf("lock por huella: %w", err)
	}

	desde := a.FirstReceivedAt.Add(-s.dedupWindow)
	var existente domain.Alert
	err = tx.QueryRowContext(ctx, `
		SELECT id, occurrences, first_received_at, coalesce(incident_id::text, '')
		FROM alerts
		WHERE fingerprint = $1 AND first_received_at >= $2
		ORDER BY first_received_at DESC
		LIMIT 1
		FOR UPDATE`, a.Fingerprint, desde).
		Scan(&existente.ID, &existente.Occurrences, &existente.FirstReceivedAt, &existente.IncidentID)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		a.ID = uuid.NewString()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO alerts (id, source, fingerprint, title, service, severity, labels, raw_payload,
			                    occurrences, incident_id, first_received_at, last_received_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,1,NULL,$9,$9)`,
			a.ID, a.Source, a.Fingerprint, a.Title, a.Service, int32(a.Severity),
			jsonMap(a.Labels), rawJSON(a.RawPayload), a.FirstReceivedAt); err != nil {
			return RecordResult{}, fmt.Errorf("insertar alerta: %w", err)
		}
		a.Occurrences = 1
		if err := insertarEvento(ctx, tx, a); err != nil {
			return RecordResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return RecordResult{}, err
		}
		return RecordResult{Alert: a, Created: true}, nil

	case err != nil:
		return RecordResult{}, fmt.Errorf("buscar por huella: %w", err)

	default:
		if _, err := tx.ExecContext(ctx, `
			UPDATE alerts
			SET occurrences = occurrences + 1,
			    last_received_at = $2,
			    severity = $3,
			    title = $4,
			    raw_payload = $5
			WHERE id = $1`,
			existente.ID, a.FirstReceivedAt, int32(a.Severity), a.Title, rawJSON(a.RawPayload)); err != nil {
			return RecordResult{}, fmt.Errorf("actualizar alerta: %w", err)
		}
		a.ID = existente.ID
		a.Occurrences = existente.Occurrences + 1
		a.FirstReceivedAt = existente.FirstReceivedAt
		a.IncidentID = existente.IncidentID
		if err := insertarEvento(ctx, tx, a); err != nil {
			return RecordResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return RecordResult{}, err
		}
		return RecordResult{Alert: a, Created: false}, nil
	}
}

func insertarEvento(ctx context.Context, tx *sql.Tx, a domain.Alert) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO alert_events (alert_id, service, received_at, payload)
		VALUES ($1, $2, $3, $4)`,
		a.ID, a.Service, a.LastReceivedAt, rawJSON(a.RawPayload)); err != nil {
		return fmt.Errorf("insertar evento: %w", err)
	}
	return nil
}

func rawJSON(b []byte) []byte {
	if len(b) == 0 || !json.Valid(b) {
		return []byte(`{}`)
	}
	return b
}

func jsonMap(m map[string]string) []byte {
	if m == nil {
		return []byte(`{}`)
	}
	b, err := json.Marshal(m)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}
