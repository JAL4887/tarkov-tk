package firestore

import (
	"context"
	"fmt"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/kyleshepherd/discord-tk-bot/internal/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const legacyImportSource = "legacy-tarkov-tk-csv"

type (
	KillStore struct {
		client *firestore.Client
	}
)

func NewKillStore(ctx context.Context, projectID string, credentialsFilePath string) (*KillStore, error) {
	conf := &firebase.Config{ProjectID: projectID}

	var app *firebase.App

	if credentialsFilePath != "" {
		var err error
		opt := option.WithCredentialsFile(credentialsFilePath)
		app, err = firebase.NewApp(ctx, conf, opt)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		app, err = firebase.NewApp(ctx, conf)
		if err != nil {
			return nil, err
		}
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	return &KillStore{
		client: client,
	}, nil
}

func (s *KillStore) GetKillByID(ctx context.Context, id string) (*storage.Kill, error) {
	dsnap, err := s.client.Collection("kills").Doc(id).Get(ctx)
	if err != nil && status.Code(err) == codes.NotFound {
		return nil, &storage.ErrNotFound{Key: id}
	}
	if err != nil {
		return nil, err
	}
	var k storage.Kill
	err = dsnap.DataTo(&k)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (s *KillStore) CreateKill(ctx context.Context, kill *storage.Kill) (*storage.Kill, error) {
	ref := s.client.Collection("kills").NewDoc()
	kill.ID = ref.ID
	_, err := ref.Set(ctx, kill)
	if err != nil {
		return nil, err
	}
	return kill, nil
}

func (s *KillStore) CreateDisappointment(ctx context.Context, disappointment *storage.Disappointment) (*storage.Disappointment, error) {
	ref := s.client.Collection("disappointments").NewDoc()
	disappointment.ID = ref.ID
	_, err := ref.Set(ctx, disappointment)
	if err != nil {
		return nil, err
	}
	return disappointment, nil
}

func (s *KillStore) DeleteKill(ctx context.Context, id string) error {
	_, err := s.client.Collection("kills").Doc(id).Delete(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *KillStore) DeleteKillsForServer(ctx context.Context, serverId string) error {
	iter := s.client.Collection("kills").Where("serverId", "==", serverId).Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		_, err = doc.Ref.Delete(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *KillStore) ListKillsForServer(ctx context.Context, serverId string) ([]*storage.Kill, error) {
	iter := s.client.Collection("kills").Where("serverId", "==", serverId).Documents(ctx)
	kills, err := iterateKills(iter)
	if err != nil {
		return nil, err
	}
	sortKillsNewestFirst(kills)
	return kills, nil
}

func (s *KillStore) ListPlayerKillsForServer(ctx context.Context, serverId string, killerId string) ([]*storage.Kill, error) {
	kills, err := s.ListKillsForServer(ctx, serverId)
	if err != nil {
		return nil, err
	}

	playerKills := make([]*storage.Kill, 0)
	for _, kill := range kills {
		if kill.Killer == killerId {
			playerKills = append(playerKills, kill)
		}
	}
	return playerKills, nil
}

func (s *KillStore) ListDisappointmentsForServer(ctx context.Context, serverId string) ([]*storage.Disappointment, error) {
	iter := s.client.Collection("disappointments").Where("serverId", "==", serverId).Documents(ctx)
	disappointments, err := iterateDisappointments(iter)
	if err != nil {
		return nil, err
	}
	sortDisappointmentsNewestFirst(disappointments)
	return disappointments, nil
}

func (s *KillStore) ListPlayerDisappointmentsForServer(ctx context.Context, serverId string, responsibleId string) ([]*storage.Disappointment, error) {
	disappointments, err := s.ListDisappointmentsForServer(ctx, serverId)
	if err != nil {
		return nil, err
	}

	playerDisappointments := make([]*storage.Disappointment, 0)
	for _, disappointment := range disappointments {
		if disappointment.Responsible == responsibleId {
			playerDisappointments = append(playerDisappointments, disappointment)
		}
	}
	return playerDisappointments, nil
}

func sortKillsNewestFirst(kills []*storage.Kill) {
	sort.SliceStable(kills, func(i, j int) bool {
		return kills[i].Date.After(kills[j].Date)
	})
}

func sortDisappointmentsNewestFirst(disappointments []*storage.Disappointment) {
	sort.SliceStable(disappointments, func(i, j int) bool {
		return disappointments[i].Date.After(disappointments[j].Date)
	})
}

func (s *KillStore) LegacyKillExists(ctx context.Context, fingerprint string) (bool, error) {
	return s.legacyRecordExists(ctx, "kills", fingerprint)
}

func (s *KillStore) LegacyDisappointmentExists(ctx context.Context, fingerprint string) (bool, error) {
	return s.legacyRecordExists(ctx, "disappointments", fingerprint)
}

func (s *KillStore) ImportLegacyKill(ctx context.Context, kill *storage.Kill, fingerprint string) (bool, error) {
	if fingerprint == "" {
		return false, fmt.Errorf("legacy fingerprint cannot be empty")
	}

	ref := s.client.Collection("kills").Doc(legacyDocumentID(fingerprint))
	kill.ID = ref.ID
	kill.LegacyImport = newLegacyImportMetadata(fingerprint)

	_, err := ref.Create(ctx, kill)
	if status.Code(err) == codes.AlreadyExists {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *KillStore) ImportLegacyDisappointment(ctx context.Context, disappointment *storage.Disappointment, fingerprint string) (bool, error) {
	if fingerprint == "" {
		return false, fmt.Errorf("legacy fingerprint cannot be empty")
	}

	ref := s.client.Collection("disappointments").Doc(legacyDocumentID(fingerprint))
	disappointment.ID = ref.ID
	disappointment.LegacyImport = newLegacyImportMetadata(fingerprint)

	_, err := ref.Create(ctx, disappointment)
	if status.Code(err) == codes.AlreadyExists {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *KillStore) legacyRecordExists(ctx context.Context, collection string, fingerprint string) (bool, error) {
	if fingerprint == "" {
		return false, fmt.Errorf("legacy fingerprint cannot be empty")
	}

	_, err := s.client.Collection(collection).Doc(legacyDocumentID(fingerprint)).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func legacyDocumentID(fingerprint string) string {
	return "legacy-" + fingerprint
}

func newLegacyImportMetadata(fingerprint string) *storage.LegacyImportMetadata {
	return &storage.LegacyImportMetadata{
		Source:      legacyImportSource,
		Fingerprint: fingerprint,
		ImportedAt:  time.Now().UTC(),
	}
}

func iterateKills(iter *firestore.DocumentIterator) ([]*storage.Kill, error) {
	kills := []*storage.Kill{}
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var k *storage.Kill
		if err := doc.DataTo(&k); err != nil {
			return nil, err
		}
		kills = append(kills, k)
	}
	return kills, nil
}

func iterateDisappointments(iter *firestore.DocumentIterator) ([]*storage.Disappointment, error) {
	disappointments := []*storage.Disappointment{}
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var disappointment *storage.Disappointment
		if err := doc.DataTo(&disappointment); err != nil {
			return nil, err
		}
		disappointments = append(disappointments, disappointment)
	}
	return disappointments, nil
}

func (s *KillStore) Close() {
	s.client.Close()
}
