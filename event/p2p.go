package event

var (
	// ETP2PGoneOnline occurs when a p2p node opens up for peer-2-peer connections
	// payload will be []multiaddr.Addr, the listening addresses of this peer
	ETP2PGoneOnline = Topic("p2p:GoneOnline")
	// ETP2PGoneOffline occurs when a p2p node has finished disconnecting from
	// a peer-2-peer network
	// payload will be nil
	ETP2PGoneOffline = Topic("p2p:GoneOffline")
	// ETP2PQriPeerConnected occurs after a qri peer has connected to this node
	// payload will be a fully hydrated *profile.Profile from
	// "github.com/qri-io/qri/repo/profile"
	ETP2PQriPeerConnected = Topic("p2p:QriPeerConnected")
	// ETP2PQriPeerDisconnected occurs after a qri peer has connected to this node
	// payload will be a fully hydrated *profile.Profile from
	// "github.com/qri-io/qri/repo/profile"
	ETP2PQriPeerDisconnected = Topic("p2p:QriPeerDisconnected")
)
