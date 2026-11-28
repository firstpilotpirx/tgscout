package main

import (
	"context"
	"fmt"

	"github.com/firstpilotpirx/tgscout/components/matching/core/application"
	"github.com/firstpilotpirx/tgscout/components/matching/core/domain"
	"github.com/firstpilotpirx/tgscout/infra/telegram"
)

// knownPeers gives the lookup the store of found chats and owners: it turns
// the lookup's KnownPeer into the matching domain's one and back. Its errors
// read as database errors, so the start reports them that way. A store call
// is local and short, so it runs to its end at a stop: a found chat is kept.
type knownPeers struct {
	repository application.PeerRepository
	path       string
}

func (k knownPeers) FindByRef(ctx context.Context, role telegram.PeerRole, ref string) (telegram.KnownPeer, bool, error) {
	peer, ok, err := k.repository.FindByRef(context.WithoutCancel(ctx), domain.PeerRole(role), ref)
	if err != nil || !ok {
		return telegram.KnownPeer{}, false, k.wrap(err)
	}
	return toTelegramPeer(peer), true, nil
}

func (k knownPeers) Save(ctx context.Context, peer telegram.KnownPeer) error {
	return k.wrap(k.repository.Save(context.WithoutCancel(ctx), toDomainPeer(peer)))
}

func (k knownPeers) DeleteByRef(ctx context.Context, role telegram.PeerRole, ref string) error {
	return k.wrap(k.repository.DeleteByRef(context.WithoutCancel(ctx), domain.PeerRole(role), ref))
}

func (k knownPeers) wrap(err error) error {
	if err == nil {
		return nil
	}
	return &storeError{path: k.path, err: err}
}

// storeError is a failed read or write of the found chats and owners.
type storeError struct {
	path string
	err  error
}

func (e *storeError) Error() string { return fmt.Sprintf("database %s: %v", e.path, e.err) }

func (e *storeError) Unwrap() error { return e.err }

func toDomainPeer(peer telegram.KnownPeer) domain.KnownPeer {
	return domain.KnownPeer{Role: domain.PeerRole(peer.Role), Ref: peer.Ref, Kind: domain.PeerKind(peer.Peer.Kind),
		PeerID: peer.Peer.ID, AccessHash: peer.Peer.AccessHash, Username: peer.Username, Title: peer.Title}
}

func toTelegramPeer(peer domain.KnownPeer) telegram.KnownPeer {
	return telegram.KnownPeer{Role: telegram.PeerRole(peer.Role), Ref: peer.Ref,
		Peer:     telegram.Peer{Kind: telegram.PeerKind(peer.Kind), ID: peer.PeerID, AccessHash: peer.AccessHash},
		Username: peer.Username, Title: peer.Title}
}
