package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"

	commonv1 "github.com/Frobama/beacon-alerts/gen/beacon/common/v1"
	"github.com/Frobama/beacon-alerts/internal/domain"
)

// Webhook es el payload de Alertmanager que nos interesa.
type Webhook struct {
	Version string  `json:"version"`
	Status  string  `json:"status"`
	Alerts  []Alert `json:"alerts"`
}

type Alert struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
	Fingerprint string            `json:"fingerprint"`
}

const maxTitulo = 255

// Normalize aplica las reglas de la Fase 2 y devuelve una alerta lista para guardar.
func Normalize(a Alert, recibido time.Time) domain.Alert {
	titulo := primeroNoVacio(a.Annotations["summary"], a.Annotations["description"], a.Labels["alertname"], "alerta sin título")
	if runas := []rune(titulo); len(runas) > maxTitulo {
		titulo = string(runas[:maxTitulo])
	}
	return domain.Alert{
		Source:          "alertmanager",
		Fingerprint:     fingerprint(a),
		Title:           titulo,
		Service:         primeroNoVacio(a.Labels["service"], a.Labels["app"], a.Labels["job"], "unknown"),
		Severity:        severidad(a.Labels["severity"]),
		Labels:          a.Labels,
		Occurrences:     1,
		FirstReceivedAt: recibido,
		LastReceivedAt:  recibido,
	}
}

func severidad(s string) commonv1.Severity {
	switch strings.ToLower(s) {
	case "critical", "page":
		return commonv1.Severity_SEVERITY_CRITICAL
	case "high", "error":
		return commonv1.Severity_SEVERITY_HIGH
	case "warning":
		return commonv1.Severity_SEVERITY_MEDIUM
	case "info", "low":
		return commonv1.Severity_SEVERITY_LOW
	default:
		// Desconocida: media, para no subestimar ni inundar de críticas.
		return commonv1.Severity_SEVERITY_MEDIUM
	}
}

func fingerprint(a Alert) string {
	if a.Fingerprint != "" {
		return a.Fingerprint
	}
	claves := make([]string, 0, len(a.Labels))
	for k := range a.Labels {
		claves = append(claves, k)
	}
	sort.Strings(claves)
	var b strings.Builder
	for _, k := range claves {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(a.Labels[k])
		b.WriteString("\n")
	}
	suma := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(suma[:])
}

func primeroNoVacio(valores ...string) string {
	for _, v := range valores {
		if v != "" {
			return v
		}
	}
	return ""
}
