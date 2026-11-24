package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/Frobama/beacon-alerts/gen/beacon/common/v1"
	"github.com/Frobama/beacon-alerts/internal/domain"
)

var ErrNotFound = errors.New("alerta no encontrada")

type ListFilter struct {
	Source    string
	Severity  commonv1.Severity
	Service   string
	PageSize  int32
	PageToken string
}

type ListResult struct {
	Alerts        []domain.Alert
	NextPageToken string
}

const (
	paginaDefault = 20
	paginaMaxima  = 100
)

const columnas = `id, source, fingerprint, title, service, severity, labels, raw_payload,
	occurrences, coalesce(incident_id::text, ''), first_received_at, last_received_at`

func (s *Store) Get(ctx context.Context, id string) (domain.Alert, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+columnas+` FROM alerts WHERE id = $1`, id)
	a, err := scanAlert(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Alert{}, ErrNotFound
	}
	return a, err
}

func (s *Store) List(ctx context.Context, f ListFilter) (ListResult, error) {
	size := f.PageSize
	if size <= 0 {
		size = paginaDefault
	}
	if size > paginaMaxima {
		size = paginaMaxima
	}

	var (
		cond = []string{"TRUE"}
		args []any
	)
	if f.Source != "" {
		args = append(args, f.Source)
		cond = append(cond, fmt.Sprintf("source = $%d", len(args)))
	}
	if f.Severity != commonv1.Severity_SEVERITY_UNSPECIFIED {
		args = append(args, int32(f.Severity))
		cond = append(cond, fmt.Sprintf("severity = $%d", len(args)))
	}
	if f.Service != "" {
		args = append(args, f.Service)
		cond = append(cond, fmt.Sprintf("service = $%d", len(args)))
	}
	if f.PageToken != "" {
		t, id, err := decodeToken(f.PageToken)
		if err != nil {
			return ListResult{}, fmt.Errorf("token de página: %w", err)
		}
		args = append(args, t, id)
		cond = append(cond, fmt.Sprintf("(last_received_at, id) < ($%d, $%d)", len(args)-1, len(args)))
	}
	args = append(args, size+1)

	query := `SELECT ` + columnas + ` FROM alerts WHERE ` + strings.Join(cond, " AND ") +
		fmt.Sprintf(` ORDER BY last_received_at DESC, id DESC LIMIT $%d`, len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()

	alertas := make([]domain.Alert, 0, size)
	for rows.Next() {
		a, err := scanAlert(rows)
		if err != nil {
			return ListResult{}, err
		}
		alertas = append(alertas, a)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, err
	}

	res := ListResult{Alerts: alertas}
	if int32(len(alertas)) > size {
		res.Alerts = alertas[:size]
		ultima := res.Alerts[size-1]
		res.NextPageToken = encodeToken(ultima.LastReceivedAt, ultima.ID)
	}
	return res, nil
}

func (s *Store) ListByIncident(ctx context.Context, incidentID string) ([]domain.Alert, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+columnas+` FROM alerts WHERE incident_id = $1 ORDER BY first_received_at ASC`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAlerts(rows)
}

func (s *Store) ListUnassigned(ctx context.Context, olderThan time.Time, limit int) ([]domain.Alert, error) {
	if limit <= 0 || limit > paginaMaxima {
		limit = paginaMaxima
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+columnas+` FROM alerts
		WHERE incident_id IS NULL AND last_received_at <= $1
		ORDER BY last_received_at ASC
		LIMIT $2`, olderThan, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAlerts(rows)
}

func (s *Store) SetIncident(ctx context.Context, alertID, incidentID string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE alerts SET incident_id = $2 WHERE id = $1 AND incident_id IS NULL`, alertID, incidentID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		// Ya estaba asociada o no existe: no es un error para el reconciliador.
		return nil
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAlert(s scanner) (domain.Alert, error) {
	var (
		a      domain.Alert
		sev    int32
		labels []byte
		raw    []byte
	)
	if err := s.Scan(&a.ID, &a.Source, &a.Fingerprint, &a.Title, &a.Service, &sev, &labels, &raw,
		&a.Occurrences, &a.IncidentID, &a.FirstReceivedAt, &a.LastReceivedAt); err != nil {
		return domain.Alert{}, err
	}
	a.Severity = commonv1.Severity(sev)
	a.Labels = decodeLabels(labels)
	a.RawPayload = raw
	return a, nil
}

func scanAlerts(rows *sql.Rows) ([]domain.Alert, error) {
	alertas := make([]domain.Alert, 0)
	for rows.Next() {
		a, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		alertas = append(alertas, a)
	}
	return alertas, rows.Err()
}

func encodeToken(t time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(t.UTC().Format(time.RFC3339Nano) + "|" + id))
}

// DecodeToken expone la decodificación del token de paginación para que otros
// paquetes (por ejemplo la API gRPC) puedan validar la entrada antes de llamar.
func DecodeToken(token string) (time.Time, string, error) {
	return decodeToken(token)
}

func decodeToken(token string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, "", err
	}
	partes := strings.SplitN(string(raw), "|", 2)
	if len(partes) != 2 {
		return time.Time{}, "", fmt.Errorf("formato inválido")
	}
	t, err := time.Parse(time.RFC3339Nano, partes[0])
	if err != nil {
		return time.Time{}, "", err
	}
	return t, partes[1], nil
}

func decodeLabels(b []byte) map[string]string {
	m := map[string]string{}
	if len(b) == 0 {
		return m
	}
	_ = jsonUnmarshal(b, &m)
	return m
}
