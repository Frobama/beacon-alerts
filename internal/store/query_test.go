package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGetAlertNoExiste(t *testing.T) {
	ctx := context.Background()
	s := nuevoStoreDeTest(t, 30*time.Minute)
	_, err := s.Get(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, quería ErrNotFound", err)
	}
}

func TestListFiltraYPagina(t *testing.T) {
	ctx := context.Background()
	s := nuevoStoreDeTest(t, 30*time.Minute)
	base := time.Now().UTC()

	for i := 0; i < 5; i++ {
		a := nuevaAlerta(string(rune('a'+i)), base.Add(time.Duration(i)*time.Minute))
		if i < 2 {
			a.Service = "otro"
		}
		if _, err := s.Record(ctx, a); err != nil {
			t.Fatal(err)
		}
	}

	pagina1, err := s.List(ctx, ListFilter{Service: "checkout", PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(pagina1.Alerts) != 2 || pagina1.NextPageToken == "" {
		t.Fatalf("página 1 = %d alertas, token %q", len(pagina1.Alerts), pagina1.NextPageToken)
	}
	pagina2, err := s.List(ctx, ListFilter{Service: "checkout", PageSize: 2, PageToken: pagina1.NextPageToken})
	if err != nil {
		t.Fatal(err)
	}
	if len(pagina2.Alerts) != 1 || pagina2.NextPageToken != "" {
		t.Fatalf("página 2 = %d alertas, token %q", len(pagina2.Alerts), pagina2.NextPageToken)
	}
}

func TestListPorIncidenteYAsociar(t *testing.T) {
	ctx := context.Background()
	s := nuevoStoreDeTest(t, 30*time.Minute)
	base := time.Now().UTC()
	incidente := "11111111-1111-1111-1111-111111111111"

	una, err := s.Record(ctx, nuevaAlerta("fp-a", base))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Record(ctx, nuevaAlerta("fp-b", base)); err != nil {
		t.Fatal(err)
	}

	if err := s.SetIncident(ctx, una.Alert.ID, incidente); err != nil {
		t.Fatal(err)
	}
	delIncidente, err := s.ListByIncident(ctx, incidente)
	if err != nil {
		t.Fatal(err)
	}
	if len(delIncidente) != 1 || delIncidente[0].ID != una.Alert.ID {
		t.Fatalf("por incidente = %+v", delIncidente)
	}

	sinIncidente, err := s.ListUnassigned(ctx, base.Add(time.Minute), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(sinIncidente) != 1 {
		t.Fatalf("sin incidente = %d, quería 1", len(sinIncidente))
	}
}
