package artifactbus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/google/uuid"
)

func TestReadURLHidesUnownedAndUnreadiness(t *testing.T) {
	owner := uuid.New()
	other := uuid.New()
	readyID := uuid.New()
	pendingID := uuid.New()
	store := &fakeStore{byID: map[uuid.UUID]Artifact{
		readyID: {
			FileID:    readyID,
			OwnerID:   owner,
			Bucket:    "cloudchamber-artifacts",
			ObjectKey: "sources/" + owner.String() + "/" + readyID.String(),
			Status:    MustParseStatus(StatusReady),
		},
		pendingID: {
			FileID:  pendingID,
			OwnerID: owner,
			Status:  Status{},
		},
	}}
	signer := &fakePresigner{url: "http://127.0.0.1:8333/signed"}
	bus := NewBusiness(nil, store, signer)

	playback, err := bus.ReadURL(context.Background(), owner, readyID)
	if err != nil {
		t.Fatalf("read owned file: %v", err)
	}
	if playback.URL != signer.url {
		t.Fatalf("url: got %q", playback.URL)
	}
	if signer.calls != 1 {
		t.Fatalf("presign calls: got %d", signer.calls)
	}

	_, err = bus.ReadURL(context.Background(), other, readyID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("other owner: got %v", err)
	}
	if signer.calls != 1 {
		t.Fatal("presign ran for an owner mismatch")
	}

	_, err = bus.ReadURL(context.Background(), owner, pendingID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("not ready: got %v", err)
	}
}

func TestFileIDFromObjectKey(t *testing.T) {
	id := uuid.MustParse("7ab08d0a-6c36-5c47-9e6d-a1ab4be4923d")
	got, err := FileIDFromObjectKey("sources/11111111-1111-4111-8111-111111111111/" + id.String())
	if err != nil {
		t.Fatal(err)
	}
	if got != id {
		t.Fatalf("got %s", got)
	}

	if _, err := FileIDFromObjectKey("sources/not-a-key"); err == nil {
		t.Fatal("expected an error")
	}
}

type fakeStore struct {
	byID map[uuid.UUID]Artifact
}

func (f *fakeStore) NewWithTx(sqldb.CommitRollbacker) (Storer, error) { return f, nil }

func (f *fakeStore) Upsert(context.Context, Artifact) error { return nil }

func (f *fakeStore) Delete(context.Context, uuid.UUID) error { return nil }

func (f *fakeStore) QueryByProject(context.Context, uuid.UUID, uuid.UUID) ([]Artifact, error) {
	return nil, nil
}

func (f *fakeStore) QueryByFileID(_ context.Context, fileID uuid.UUID) (Artifact, error) {
	art, ok := f.byID[fileID]
	if !ok {
		return Artifact{}, ErrNotFound
	}
	return art, nil
}

func (f *fakeStore) QueryByFileIDs(context.Context, []uuid.UUID) ([]Artifact, error) {
	return nil, nil
}

type fakePresigner struct {
	url   string
	calls int
}

func (f *fakePresigner) PresignGet(context.Context, string, string, time.Duration) (string, time.Time, error) {
	f.calls++
	return f.url, time.Now().Add(readTTL), nil
}
