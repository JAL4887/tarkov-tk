package migration

import (
	"strings"
	"testing"
	"time"
)

func TestParseLegacyCSV(t *testing.T) {
	csvData := `Killer,Victim,Reason,Date
alpha,bravo,normal TK,2026-08-12T15:52:03.213428Z
charlie,delta,,2026-08-12T15:46:04.887857Z
`

	records, err := ParseLegacyCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("ParseLegacyCSV returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Killer != "alpha" || records[0].Victim != "bravo" || records[0].Reason != "normal TK" {
		t.Fatalf("unexpected first record: %+v", records[0])
	}
	if records[1].Reason != "" {
		t.Fatalf("expected empty reason, got %q", records[1].Reason)
	}
}

func TestIsDisappointmentIsCaseInsensitive(t *testing.T) {
	for _, reason := range []string{
		"disappointment",
		"The infamous green flare disappointment",
		"DISAPPOINTMENT",
		"Absolute Disappointment in the hallway",
	} {
		if !IsDisappointment(reason) {
			t.Fatalf("expected %q to classify as disappointment", reason)
		}
	}

	if IsDisappointment("bad grenade") {
		t.Fatal("normal TK reason should not classify as disappointment")
	}
}

func TestFingerprintIsStableAndSensitiveToRecordData(t *testing.T) {
	date := time.Date(2026, time.August, 12, 15, 46, 4, 887857000, time.UTC)
	record := LegacyRecord{
		Killer: "a_wisp",
		Victim: "jimjamjub",
		Reason: "The infamous green flare disappointment",
		Date:   date,
	}

	first := Fingerprint(record)
	second := Fingerprint(record)
	if first != second {
		t.Fatalf("fingerprint must be stable: %q != %q", first, second)
	}

	record.Reason = "different reason"
	if Fingerprint(record) == first {
		t.Fatal("fingerprint should change when record data changes")
	}
}
