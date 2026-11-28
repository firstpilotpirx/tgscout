//go:build integration

package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firstpilotpirx/tgscout/components/matching/adapters/secondary/persistence/sqlite"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
)

func newPeerRepository(t *testing.T) *sqlite.SqlitePeerRepository {
	t.Helper()
	db, _ := openTemp(t)
	_, err := sqlite.Migrate(db)
	require.NoError(t, err)
	return sqlite.NewSqlitePeerRepository(db)
}

var madeUpChannel = domain.KnownPeer{Role: domain.PeerChat, Ref: "@made_up_market", Kind: domain.PeerKindChannel,
	PeerID: 9001, AccessHash: 424242, Username: "made_up_market", Title: "Made-up market"}

func TestPeerRepositoryFindsWhatItSaved(t *testing.T) {
	repository := newPeerRepository(t)
	ctx := context.Background()

	require.NoError(t, repository.Save(ctx, madeUpChannel))
	got, ok, err := repository.FindByRef(ctx, domain.PeerChat, "@made_up_market")

	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, madeUpChannel, got)
}

func TestPeerRepositoryFindsNothingForAnUnknownRef(t *testing.T) {
	repository := newPeerRepository(t)

	_, ok, err := repository.FindByRef(context.Background(), domain.PeerChat, "@made_up_nothing")

	require.NoError(t, err)
	assert.False(t, ok)
}

func TestPeerRepositoryKeepsAChatAndAnOwnerOfOneRefApart(t *testing.T) {
	repository := newPeerRepository(t)
	ctx := context.Background()
	owner := domain.KnownPeer{Role: domain.PeerOwner, Ref: "@made_up_market", Kind: domain.PeerKindUser,
		PeerID: 77, AccessHash: 5, Username: "made_up_market"}
	require.NoError(t, repository.Save(ctx, madeUpChannel))
	require.NoError(t, repository.Save(ctx, owner))

	require.NoError(t, repository.DeleteByRef(ctx, domain.PeerChat, "@made_up_market"))

	_, chatKept, err := repository.FindByRef(ctx, domain.PeerChat, "@made_up_market")
	require.NoError(t, err)
	got, ownerKept, err := repository.FindByRef(ctx, domain.PeerOwner, "@made_up_market")
	require.NoError(t, err)
	assert.False(t, chatKept)
	assert.True(t, ownerKept)
	assert.Equal(t, owner, got)
}

func TestPeerRepositoryReplacesOnASecondSave(t *testing.T) {
	repository := newPeerRepository(t)
	ctx := context.Background()
	require.NoError(t, repository.Save(ctx, madeUpChannel))
	moved := madeUpChannel
	moved.PeerID, moved.Title = 9002, "Made-up market, new title"

	require.NoError(t, repository.Save(ctx, moved))
	got, ok, err := repository.FindByRef(ctx, domain.PeerChat, "@made_up_market")

	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, moved, got)
}

func TestPeerRepositoryDeletingAnUnknownRefIsNoError(t *testing.T) {
	repository := newPeerRepository(t)

	assert.NoError(t, repository.DeleteByRef(context.Background(), domain.PeerOwner, "100200300"))
}
