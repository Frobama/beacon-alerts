package ingest

import (
	"strings"
	"testing"
	"time"

	commonv1 "github.com/Frobama/beacon-alerts/gen/beacon/common/v1"
)

var recibido = time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)

func TestTituloUsaSummaryLuegoDescriptionLuegoAlertname(t *testing.T) {
	casos := []struct {
		nombre string
		in     Alert
		want   string
	}{
		{"summary", Alert{Labels: map[string]string{"alertname": "A"}, Annotations: map[string]string{"summary": "S", "description": "D"}}, "S"},
		{"description", Alert{Labels: map[string]string{"alertname": "A"}, Annotations: map[string]string{"description": "D"}}, "D"},
		{"alertname", Alert{Labels: map[string]string{"alertname": "A"}}, "A"},
		{"vacío", Alert{}, "alerta sin título"},
	}
	for _, c := range casos {
		if got := Normalize(c.in, recibido).Title; got != c.want {
			t.Fatalf("%s: título = %q, quería %q", c.nombre, got, c.want)
		}
	}
}

func TestTituloSeTruncaA255Caracteres(t *testing.T) {
	largo := strings.Repeat("á", 300)
	a := Normalize(Alert{Annotations: map[string]string{"summary": largo}}, recibido)
	if len([]rune(a.Title)) != 255 {
		t.Fatalf("largo = %d runas, quería 255", len([]rune(a.Title)))
	}
}

func TestServicioUsaServiceLuegoAppLuegoJob(t *testing.T) {
	casos := []struct {
		labels map[string]string
		want   string
	}{
		{map[string]string{"service": "checkout", "app": "x", "job": "y"}, "checkout"},
		{map[string]string{"app": "x", "job": "y"}, "x"},
		{map[string]string{"job": "y"}, "y"},
		{map[string]string{}, "unknown"},
	}
	for _, c := range casos {
		if got := Normalize(Alert{Labels: c.labels}, recibido).Service; got != c.want {
			t.Fatalf("%v: servicio = %q, quería %q", c.labels, got, c.want)
		}
	}
}

func TestSeveridad(t *testing.T) {
	casos := map[string]commonv1.Severity{
		"critical": commonv1.Severity_SEVERITY_CRITICAL,
		"page":     commonv1.Severity_SEVERITY_CRITICAL,
		"high":     commonv1.Severity_SEVERITY_HIGH,
		"error":    commonv1.Severity_SEVERITY_HIGH,
		"warning":  commonv1.Severity_SEVERITY_MEDIUM,
		"info":     commonv1.Severity_SEVERITY_LOW,
		"low":      commonv1.Severity_SEVERITY_LOW,
		"":         commonv1.Severity_SEVERITY_MEDIUM,
		"raro":     commonv1.Severity_SEVERITY_MEDIUM,
	}
	for in, want := range casos {
		got := Normalize(Alert{Labels: map[string]string{"severity": in}}, recibido).Severity
		if got != want {
			t.Fatalf("%q: severidad = %v, quería %v", in, got, want)
		}
	}
}

func TestFingerprintPropioGanaYSiFaltaEsEstable(t *testing.T) {
	a := Normalize(Alert{Fingerprint: "abc", Labels: map[string]string{"b": "2", "a": "1"}}, recibido)
	if a.Fingerprint != "abc" {
		t.Fatalf("fingerprint = %q", a.Fingerprint)
	}
	uno := Normalize(Alert{Labels: map[string]string{"a": "1", "b": "2"}}, recibido).Fingerprint
	dos := Normalize(Alert{Labels: map[string]string{"b": "2", "a": "1"}}, recibido).Fingerprint
	if uno != dos || len(uno) != 64 {
		t.Fatalf("fingerprints = %q / %q", uno, dos)
	}
}
