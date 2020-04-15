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
	// ETP2PPeerConnected occurs after any peer has connected to this node
	// payload will be a libp2p.peerInfo
	ETP2PPeerConnected = Topic("p2p:PeerConnected")
	// ETP2PPeerDisconnected occurs after any peer has connected to this node
	// payload will be a libp2p.peerInfo
	ETP2PPeerDisconnected = Topic("p2p:PeerDisconnected")
	// ETP2PMessageReceived fires whenever the p2p protocol receives a message
	// from a Qri peer
	// payload will be a p2p.Message
	ETP2PMessageReceived = Topic("p2p:MessageReceived")
)
