package latencyprotocol

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3/log"
	sigAlg "golang.org/x/crypto/ed25519"
	"testing"
	"time"
)

func TestBlacklistOnAccurateChainEmpty(t *testing.T) {

	N := 4
	x := 4
	d := 1 * time.Nanosecond
	suspicionThreshold := 0

	chain, nodeIDs := initChain(N, x, accurate, 0, 0)

	for _, NodeID := range nodeIDs {
		node := Node{
			ID:                      NodeID,
			SendingAddress:          "address",
			PrivateKey:              nil,
			LatenciesInConstruction: nil,
			BlockSkeleton:           nil,
			NbLatenciesRefreshed:    0,
			IncomingMessageChannel:  nil,
			BlockChannel:            nil,
		}

		blacklist, err := node.CreateBlacklist(chain, d, suspicionThreshold)

		require.NoError(t, err)
		require.Zero(t, blacklist.Size(), "Blacklist should be empty")

	}
}

func TestBlacklistOnInaccurateChainAllBlacklisted(t *testing.T) {

	N := 4
	x := 4
	d := 1 * time.Nanosecond
	suspicionThreshold := 1

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := initChain(N, x, inaccurate, N, N)

	for index, NodeID := range nodeIDs {
		node := Node{
			ID:                      NodeID,
			SendingAddress:          "address",
			PrivateKey:              nil,
			LatenciesInConstruction: nil,
			BlockSkeleton:           nil,
			NbLatenciesRefreshed:    0,
			IncomingMessageChannel:  nil,
			BlockChannel:            nil,
		}

		blacklist, err := node.CreateBlacklist(chain, d, suspicionThreshold)

		blacklists[index] = blacklist

		require.NoError(t, err)
		require.Equal(t, N, blacklist.Size(), "Blacklist should contain all nodes")

	}
}

func TestBlacklistOneLiarOneVictim(t *testing.T) {
	N := 4
	d := 1 * time.Nanosecond
	suspicionThreshold := 0

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := fourNodeChain()

	setLiarAndVictim(chain, "A", "D", 25)

	for index, key := range nodeIDs {
		node := Node{
			ID:                      &NodeID{nil, key},
			SendingAddress:          "address",
			PrivateKey:              nil,
			LatenciesInConstruction: nil,
			BlockSkeleton:           nil,
			NbLatenciesRefreshed:    0,
			IncomingMessageChannel:  nil,
			BlockChannel:            nil,
		}

		blacklist, err := node.CreateBlacklist(chain, d, suspicionThreshold)

		blacklists[index] = blacklist

		require.NoError(t, err)

	}

	firstBlacklist := blacklists[0]

	require.Equal(t, N, firstBlacklist.Size())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

	//expect both liar and victim to get 2 strikes
	require.Equal(t, 2, strikes[1])
	require.Equal(t, 2, strikes[2])

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}

func TestBlacklistOneLiarTwoVictims(t *testing.T) {
	N := 4
	d := 1 * time.Nanosecond
	suspicionThreshold := 0

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := fourNodeChain()

	setLiarAndVictim(chain, "A", "B", 25)
	setLiarAndVictim(chain, "A", "C", 25)

	for index, key := range nodeIDs {
		node := Node{
			ID:                      &NodeID{nil, key},
			SendingAddress:          "address",
			PrivateKey:              nil,
			LatenciesInConstruction: nil,
			BlockSkeleton:           nil,
			NbLatenciesRefreshed:    0,
			IncomingMessageChannel:  nil,
			BlockChannel:            nil,
		}

		blacklist, err := node.CreateBlacklist(chain, d, suspicionThreshold)

		blacklists[index] = blacklist

		require.NoError(t, err)

	}

	firstBlacklist := blacklists[0]

	require.Equal(t, N, firstBlacklist.Size())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

	//expect both liar and non-victim to get 2 strikes
	require.Equal(t, 2, strikes[1])
	require.Equal(t, 2, strikes[2])

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}

func TestBlacklistSmallNetworkAssimmetricalLies(t *testing.T) {

	// A <-> D does not make sense, not enough info to know who is evil

	N := 4
	d := 1 * time.Nanosecond
	suspicionThreshold := 1

	chain, nodes := fourNodeChain()

	chain.Blocks[0].Latencies["D"] = ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[3].Latencies["A"] = ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[1].Latencies["D"] = ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[3].Latencies["B"] = ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil}

	blacklists := make([]Blacklistset, N)

	for index, key := range nodes {
		node := Node{
			ID:                      &NodeID{nil, key},
			SendingAddress:          "address",
			PrivateKey:              nil,
			LatenciesInConstruction: nil,
			BlockSkeleton:           nil,
			NbLatenciesRefreshed:    0,
			IncomingMessageChannel:  nil,
			BlockChannel:            nil,
		}

		blacklist, err := node.CreateBlacklist(chain, d, suspicionThreshold)

		require.NoError(t, err)
		require.NotZero(t, blacklist.Size())
		blacklists[index] = blacklist

	}

	firstBlacklist := blacklists[0]

	require.Equal(t, 2, firstBlacklist.Size())
	require.True(t, firstBlacklist.Contains(sigAlg.PublicKey([]byte("A")), 0))
	require.True(t, firstBlacklist.Contains(sigAlg.PublicKey([]byte("D")), 0))

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}

func TestBlacklistExactlyOneLiarLargeAssimmetricalLies(t *testing.T) {

	N := 5
	d := 1 * time.Nanosecond
	suspicionThreshold := 2

	chain, nodes := fiveNodeChain()

	chain.Blocks[1].Latencies["C"] = ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[1].Latencies["D"] = ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil}

	chain.Blocks[2].Latencies["B"] = ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[2].Latencies["D"] = ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[2].Latencies["E"] = ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil}

	chain.Blocks[3].Latencies["B"] = ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[3].Latencies["C"] = ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[3].Latencies["E"] = ConfirmedLatency{time.Duration(20 * time.Nanosecond), nil, time.Now(), nil}

	chain.Blocks[4].Latencies["C"] = ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[4].Latencies["D"] = ConfirmedLatency{time.Duration(20 * time.Nanosecond), nil, time.Now(), nil}

	blacklists := make([]Blacklistset, N)

	for index, key := range nodes {
		node := Node{
			ID:                      &NodeID{nil, key},
			SendingAddress:          "address",
			PrivateKey:              nil,
			LatenciesInConstruction: nil,
			BlockSkeleton:           nil,
			NbLatenciesRefreshed:    0,
			IncomingMessageChannel:  nil,
			BlockChannel:            nil,
		}

		blacklist, err := node.CreateBlacklist(chain, d, suspicionThreshold)

		require.NoError(t, err)
		require.NotZero(t, blacklist.Size())
		blacklists[index] = blacklist

	}

	firstBlacklist := blacklists[0]

	require.Equal(t, 1, firstBlacklist.Size())
	require.True(t, firstBlacklist.Contains(sigAlg.PublicKey([]byte("C")), suspicionThreshold))

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}

/*func blacklistsEquivalent(a, b []sigAlg.PublicKey) bool {

	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if !contains(b, a[i]) {
			return false
		}
	}

	return true
}

func contains(s []sigAlg.PublicKey, e sigAlg.PublicKey) bool {
	for _, a := range s {
		if reflect.DeepEqual(a, e) {
			return true
		}
	}
	return false
}*/
