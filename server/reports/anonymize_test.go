package reports

import (
	"testing"
	"time"
)

func TestAnonymizePhotosSuppressesSmallBuckets(t *testing.T) {
	photos := []photoDoc{
		{ProfesieFunctie: "Medic", Data: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), TipControl: "Periodic", AvizMedical: "APT"},
		{ProfesieFunctie: "Medic", Data: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), TipControl: "Periodic", AvizMedical: "APT"},
		{ProfesieFunctie: "Sofer", Data: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), TipControl: "Angajare", AvizMedical: "APT"},
	}
	rows := anonymizePhotos(photos, 5)
	if len(rows) != 1 {
		t.Fatalf("expected one suppressed row, got %d", len(rows))
	}
	if rows[0]["profession"] != "suppressed" {
		t.Fatalf("expected suppressed profession, got %v", rows[0]["profession"])
	}
	if rows[0]["documents"] != 3 {
		t.Fatalf("expected 3 documents, got %v", rows[0]["documents"])
	}
}
