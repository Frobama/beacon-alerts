package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Frobama/beacon-alerts/internal/ingest"
)

const maxCuerpo = 1 << 20 // 1 MB

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if !s.autorizado(r) {
		http.Error(w, "no autorizado", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxCuerpo)
	var wh ingest.Webhook
	if err := json.NewDecoder(r.Body).Decode(&wh); err != nil {
		var demasiado *http.MaxBytesError
		if errors.As(err, &demasiado) {
			http.Error(w, "cuerpo demasiado grande", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "cuerpo inválido", http.StatusBadRequest)
		return
	}

	recibido := time.Now().UTC()
	aceptadas := 0
	for _, a := range wh.Alerts {
		if strings.EqualFold(a.Status, "resolved") {
			s.resueltas.Add(1)
			s.log.Info("alerta resuelta ignorada", "fingerprint", a.Fingerprint)
			continue
		}
		normalizada := ingest.Normalize(a, recibido)
		normalizada.RawPayload = nil

		res, err := s.store.Record(r.Context(), normalizada)
		if err != nil {
			s.log.Error("guardar alerta", "err", err)
			http.Error(w, "no se pudo guardar", http.StatusInternalServerError)
			return
		}
		aceptadas++
		if !res.Created {
			continue
		}

		incidentID, _, err := s.reporter.ReportAlert(r.Context(), res.Alert)
		if err != nil {
			s.log.Warn("reportar a incidentes", "alert_id", res.Alert.ID, "err", err)
			continue // queda sin incidente; el reconciliador lo reintenta
		}
		if err := s.store.SetIncident(r.Context(), res.Alert.ID, incidentID); err != nil {
			s.log.Warn("asociar incidente", "alert_id", res.Alert.ID, "err", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]int{"accepted": aceptadas})
}

func (s *Server) autorizado(r *http.Request) bool {
	if s.cfg.WebhookToken == "" {
		return true
	}
	header := r.Header.Get("Authorization")
	const prefijo = "Bearer "
	if !strings.HasPrefix(header, prefijo) {
		return false
	}
	got := []byte(strings.TrimPrefix(header, prefijo))
	want := []byte(s.cfg.WebhookToken)
	return subtle.ConstantTimeCompare(got, want) == 1
}

var _ = io.Discard
