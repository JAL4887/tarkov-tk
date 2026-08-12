package migration

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"
)

type LegacyRecord struct {
	Killer string
	Victim string
	Reason string
	Date   time.Time
}

func ParseLegacyCSV(r io.Reader) ([]LegacyRecord, error) {
	reader := csv.NewReader(r)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}

	indexes := make(map[string]int, len(header))
	for i, value := range header {
		indexes[strings.ToLower(strings.TrimSpace(value))] = i
	}

	for _, required := range []string{"killer", "victim", "reason", "date"} {
		if _, ok := indexes[required]; !ok {
			return nil, fmt.Errorf("CSV is missing required column %q", required)
		}
	}

	records := []LegacyRecord{}
	rowNumber := 1
	for {
		rowNumber++
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV row %d: %w", rowNumber, err)
		}

		killer, err := valueAt(row, indexes["killer"])
		if err != nil {
			return nil, fmt.Errorf("row %d killer: %w", rowNumber, err)
		}
		victim, err := valueAt(row, indexes["victim"])
		if err != nil {
			return nil, fmt.Errorf("row %d victim: %w", rowNumber, err)
		}
		reason, err := valueAt(row, indexes["reason"])
		if err != nil {
			return nil, fmt.Errorf("row %d reason: %w", rowNumber, err)
		}
		dateValue, err := valueAt(row, indexes["date"])
		if err != nil {
			return nil, fmt.Errorf("row %d date: %w", rowNumber, err)
		}

		killer = strings.TrimSpace(killer)
		victim = strings.TrimSpace(victim)
		if killer == "" || victim == "" {
			return nil, fmt.Errorf("row %d must include both killer and victim", rowNumber)
		}

		date, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(dateValue))
		if err != nil {
			return nil, fmt.Errorf("row %d has invalid date %q: %w", rowNumber, dateValue, err)
		}

		records = append(records, LegacyRecord{
			Killer: killer,
			Victim: victim,
			Reason: reason,
			Date:   date,
		})
	}

	return records, nil
}

func IsDisappointment(reason string) bool {
	return strings.Contains(strings.ToLower(reason), "disappointment")
}

func Fingerprint(record LegacyRecord) string {
	canonical := strings.Join([]string{
		strings.TrimSpace(record.Killer),
		strings.TrimSpace(record.Victim),
		record.Reason,
		record.Date.UTC().Format(time.RFC3339Nano),
	}, "\x00")

	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

func valueAt(row []string, index int) (string, error) {
	if index < 0 || index >= len(row) {
		return "", fmt.Errorf("column index %d is outside row with %d columns", index, len(row))
	}
	return row[index], nil
}
