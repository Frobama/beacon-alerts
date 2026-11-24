package store

import (
	"context"
	"sync"
	"testing"
	"time"

	commonv1 "github.com/Frobama/beacon-alerts/gen/beacon/common/v1"
	"github.com/Frobama/beacon-alerts/internal/domain"
)

func nuevaAlerta(fp string, recibido time.Time) domain.Alert {
	return domain.Alert{
		Source:          "alertmanager",
		Fingerprint:     fp,
		Title:           "título",
		Service:         "checkout",
		Severity:        commonv1.Severity_SEVERITY_HIGH,
		Labels:          map[string]string{"service": "checkout"},
		RawPayload:      []byte(`{"status":"firing"}`),
		Occurrences:     1,
		FirstReceivedAt: recibido,
		LastReceivedAt:  recibido,
	}
}

func TestRecordCreaAlertaYEvento(t *testing.T) {
	ctx := context.Background()
	s := nuevoStoreDeTest(t, 30*time.Minute)
	recibido := time.Now().UTC()

	res, err := s.Record(ctx, nuevaAlerta("fp-1", recibido))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created || res.Alert.ID == "" || res.Alert.Occurrences != 1 {
		t.Fatalf("resultado = %+v", res)
	}

	var eventos int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM alert_events WHERE alert_id = $1`, res.Alert.ID).Scan(&eventos); err != nil {
		t.Fatal(err)
	}
	if eventos != 1 {
		t.Fatalf("eventos = %d, quería 1", eventos)
	}
}

func TestRecordDeduplicaEnVentana(t *testing.T) {
	ctx := context.Background()
	s := nuevoStoreDeTest(t, 30*time.Minute)
	base := time.Now().UTC()

	primera, err := s.Record(ctx, nuevaAlerta("fp-2", base))
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := s.Record(ctx, nuevaAlerta("fp-2", base.Add(10*time.Minute)))
	if err != nil {
		t.Fatal(err)
	}

	if segunda.Created {
		t.Fatal("la segunda no debía crear otra alerta")
	}
	if segunda.Alert.ID != primera.Alert.ID || segunda.Alert.Occurrences != 2 {
		t.Fatalf("segunda = %+v", segunda.Alert)
	}

	var filas int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM alerts WHERE fingerprint = 'fp-2'`).Scan(&filas); err != nil {
		t.Fatal(err)
	}
	if filas != 1 {
		t.Fatalf("filas = %d, quería 1", filas)
	}
}

func TestRecordFueraDeVentanaCreaOtra(t *testing.T) {
	ctx := context.Background()
	s := nuevoStoreDeTest(t, 30*time.Minute)
	base := time.Now().UTC()

	if _, err := s.Record(ctx, nuevaAlerta("fp-3", base)); err != nil {
		t.Fatal(err)
	}
	otra, err := s.Record(ctx, nuevaAlerta("fp-3", base.Add(31*time.Minute)))
	if err != nil {
		t.Fatal(err)
	}
	if !otra.Created {
		t.Fatal("fuera de la ventana debía crear una alerta nueva")
	}
}

func TestRecordConcurrenteNoDuplica(t *testing.T) {
	ctx := context.Background()
	s := nuevoStoreDeTest(t, 30*time.Minute)
	base := time.Now().UTC()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Record(ctx, nuevaAlerta("fp-4", base)); err != nil {
				t.Errorf("record: %v", err)
			}
		}()
	}
	wg.Wait()

	var filas, ocurrencias int
	if err := s.db.QueryRowContext(ctx,
		`SELECT count(*), coalesce(max(occurrences), 0) FROM alerts WHERE fingerprint = 'fp-4'`).Scan(&filas, &ocurrencias); err != nil {
		t.Fatal(err)
	}
	if filas != 1 || ocurrencias != 8 {
		t.Fatalf("filas = %d, ocurrencias = %d", filas, ocurrencias)
	}
}
