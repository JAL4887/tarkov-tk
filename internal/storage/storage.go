package storage

import (
	"context"
	"fmt"
	"time"
)

type KillStore interface {
	KillGetter
	KillPutter
	KillDeleter
	KillLister
	DisappointmentPutter
	LegacyImporter
	KillCloser
}

type KillGetter interface {
	GetKillByID(ctx context.Context, id string) (*Kill, error)
}

type KillPutter interface {
	CreateKill(ctx context.Context, kill *Kill) (*Kill, error)
}

type KillDeleter interface {
	DeleteKill(ctx context.Context, id string) error
	DeleteKillsForServer(ctx context.Context, serverId string) error
}

type KillLister interface {
	ListKillsForServer(ctx context.Context, serverId string) ([]*Kill, error)
	ListPlayerKillsForServer(ctx context.Context, serverId string, killerId string) ([]*Kill, error)
}

type DisappointmentPutter interface {
	CreateDisappointment(ctx context.Context, disappointment *Disappointment) (*Disappointment, error)
}

type LegacyImporter interface {
	LegacyKillExists(ctx context.Context, fingerprint string) (bool, error)
	LegacyDisappointmentExists(ctx context.Context, fingerprint string) (bool, error)
	ImportLegacyKill(ctx context.Context, kill *Kill, fingerprint string) (bool, error)
	ImportLegacyDisappointment(ctx context.Context, disappointment *Disappointment, fingerprint string) (bool, error)
}

type KillCloser interface {
	Close()
}

type LegacyImportMetadata struct {
	Source      string    `firestore:"source"`
	Fingerprint string    `firestore:"fingerprint"`
	ImportedAt  time.Time `firestore:"importedAt"`
}

type Kill struct {
	ID           string                `firestore:"id" csv:"-"`
	ServerID     string                `firestore:"serverId" csv:"-"`
	Killer       string                `firestore:"killer"`
	Victim       string                `firestore:"victim"`
	Reason       string                `firestore:"reason"`
	Date         time.Time             `firestore:"date"`
	LegacyImport *LegacyImportMetadata `firestore:"legacyImport,omitempty" csv:"-"`
}

type Disappointment struct {
	ID           string                `firestore:"id" csv:"-"`
	ServerID     string                `firestore:"serverId" csv:"-"`
	Responsible  string                `firestore:"responsible"`
	Victim       string                `firestore:"victim"`
	Reason       string                `firestore:"reason"`
	Date         time.Time             `firestore:"date"`
	LegacyImport *LegacyImportMetadata `firestore:"legacyImport,omitempty" csv:"-"`
}

type ErrNotFound struct {
	Key string
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("Kill not found: %s", e.Key)
}
