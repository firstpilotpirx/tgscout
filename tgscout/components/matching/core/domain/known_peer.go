package domain

// PeerRole tells what a stored peer is to tgscout: a watched chat or the owner.
type PeerRole string

// The roles of a stored peer.
const (
	PeerChat  PeerRole = "chat"
	PeerOwner PeerRole = "owner"
)

// PeerKind is the Telegram kind of a stored peer.
type PeerKind string

// The kinds of a stored peer: a channel or supergroup, a basic group, a user.
const (
	PeerKindChannel PeerKind = "channel"
	PeerKindChat    PeerKind = "chat"
	PeerKindUser    PeerKind = "user"
)

// KnownPeer is a chat or the owner tgscout found once, kept by its config
// text so a restart needs no lookup by name. AccessHash is the per-account
// value Telegram gives for the peer; 0 for a basic group. It holds no message text.
type KnownPeer struct {
	Role       PeerRole
	Ref        string
	Kind       PeerKind
	PeerID     int64
	AccessHash int64
	Username   string
	Title      string
}
