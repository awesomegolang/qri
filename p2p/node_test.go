package p2p

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/qri-io/qri/config"
	cfgtest "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/event"
	p2ptest "github.com/qri-io/qri/p2p/test"
	"github.com/qri-io/qri/repo/profile"
	"github.com/qri-io/qri/repo/test"
)

func TestNewNode(t *testing.T) {
	ctx, done := context.WithCancel(context.Background())
	defer done()

	info := cfgtest.GetTestPeerInfo(0)
	r, err := test.NewTestRepoFromProfileID(profile.IDFromPeerID(info.PeerID), 0, -1)
	if err != nil {
		t.Errorf("error creating test repo: %s", err.Error())
		return
	}

	p2pconf := config.DefaultP2PForTesting()
	node, err := NewTestableQriNode(r, p2pconf)
	if err != nil {
		t.Errorf("error creating qri node: %s", err.Error())
		return
	}
	n := node.(*QriNode)
	if n.Online {
		t.Errorf("default node online flag should be false")
	}
	if err := n.GoOnline(ctx); err != nil {
		t.Error(err.Error())
	}
	if !n.Online {
		t.Errorf("online should equal true")
	}
}

func TestNodeEvents(t *testing.T) {
	var (
		bus    event.Bus
		result = make(chan error)
		events = []event.Topic{
			// TODO (b5) - can't check onlineness because of the way this test is constructed
			// event.ETP2PGoneOnline,
			event.ETP2PGoneOffline,
			event.ETP2PQriPeerConnected,
			// TODO (b5) - this event currently isn't emitted
			// event.ETP2PQriPeerDisconnected,
			event.ETP2PPeerConnected,
			event.ETP2PPeerDisconnected,
		}
	)

	ctx, done := context.WithTimeout(context.Background(), time.Second)
	defer done()

	bus = event.NewBus(ctx)
	ch := bus.Subscribe(events...)
	called := map[event.Topic]bool{}
	for _, t := range events {
		called[t] = false
	}

	go func() {
		remaining := len(events)
		for {
			select {
			case e := <-ch:
				if called[e.Topic] {
					t.Errorf("expected event %q to only fire once", e.Topic)
				}

				called[e.Topic] = true
				remaining--
				if remaining == 0 {
					result <- nil
					return
				}
			case <-ctx.Done():
				ok := true
				uncalled := ""
				for topic, called := range called {
					if !called {
						uncalled += fmt.Sprintf("%s\n", topic)
						ok = false
					}
				}
				if !ok {
					result <- fmt.Errorf("context cancelled before all events could fire. Uncalled Events:\n%s", uncalled)
					return
				}
				result <- nil
			}
		}
	}()

	factory := p2ptest.NewTestNodeFactory(NewTestableQriNode)
	testPeers, err := p2ptest.NewTestNetwork(ctx, factory, 2)
	if err != nil {
		t.Fatalf("error creating network: %s", err.Error())
	}
	peers := asQriNodes(testPeers)
	peers[0].pub = bus

	if err := p2ptest.ConnectQriNodes(ctx, testPeers); err != nil {
		t.Fatalf("error connecting peers: %s", err.Error())
	}

	if err := peers[1].GoOffline(); err != nil {
		t.Error(err)
	}

	if err := peers[0].GoOffline(); err != nil {
		t.Error(err)
	}

	if err := <-result; err != nil {
		t.Error(err)
	}
}
