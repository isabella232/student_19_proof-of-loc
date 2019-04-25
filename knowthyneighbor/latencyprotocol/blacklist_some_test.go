package latencyprotocol

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3/log"
	sigAlg "golang.org/x/crypto/ed25519"
	"testing"
	"time"
)

func TestBlacklistStillAccurateOnEmptyChain(t *testing.T) {

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

func TestBlacklistStillInaccurateAllBlacklisted(t *testing.T) {

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

func TestBlacklistOneLiarOneVictimOneUnknown(t *testing.T) {
	N := 4
	d := 1 * time.Nanosecond
	suspicionThreshold := 0

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "A", "D", 25)
	deleteLatency(chain, "A", "B")
	//deleteLatency(chain, "A", "C")
	//deleteLatency(chain, "B", "C")
	//deleteLatency(chain, "B", "D")
	//deleteLatency(chain, "C", "D")

	/*
	* Delete A-B -> A,C,D: 1 strike
	* Delete A-C -> A,B,D: 1 strike
	* Delete B-C -> No change
	* Delete B-D -> A,C,D: 1 strike
	* Delete C-D -> A,B,D: 1 strike
	* Delete A-B and A-C: no strikes
	* Delete A-B and C-D:No strikes
	* Delete A-C and B-D: No strikes
	* Delete B-D and C-D: No strikes


	 */

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

	log.Print(firstBlacklist.ToString())

	//require.Equal(t, N, firstBlacklist.Size())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

}

func TestBlacklistOneLiarTwoVictimsSomeUnknown(t *testing.T) {
	N := 4
	d := 1 * time.Nanosecond
	suspicionThreshold := 0

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "A", "B", 25)
	setLiarAndVictim(chain, "A", "C", 25)
	//deleteLatency(chain, "A", "D")
	deleteLatency(chain, "B", "C")
	//deleteLatency(chain, "B", "D")
	//deleteLatency(chain, "C", "D")

	/**
	* Delete A-D: no strikes
	* Delete B-C: no changes
	* Delete B-D: A,C,D: 1 strike
	* Delete C-D: A,B,D: 1 strike

	**/

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
	log.Print(firstBlacklist.ToString())

	//require.Equal(t, N, firstBlacklist.Size())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

}

func TestBlacklistFiveNodesOneLiarTwoVictimsSomeUnknown(t *testing.T) {
	N := 5
	d := 1 * time.Nanosecond
	suspicionThreshold := 0

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "E", "A", 25)
	setLiarAndVictim(chain, "E", "B", 25)
	//deleteLatency(chain, "A", "B") //same
	//deleteLatency(chain, "A", "C") //1 less strike, still works
	//deleteLatency(chain, "A", "D") //1 less strike, still works
	//deleteLatency(chain, "B", "C") //1 less strike, still works
	//deleteLatency(chain, "B", "D") //1 less strike, still works
	//deleteLatency(chain, "C", "D") //same
	//deleteLatency(chain, "C", "E") //D and E both blacklisted
	//deleteLatency(chain, "D", "E") //D and E both blacklisted

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

	firstBlacklist := blacklists[1]
	log.Print(firstBlacklist.ToString())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

}

func TestBlacklistEightNodesTwoLiarsThreeVictimsSomeUnknown(t *testing.T) {
	N := 8
	d := 1 * time.Nanosecond
	suspicionThreshold := 0

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "G", "A", 25)
	setLiarAndVictim(chain, "G", "B", 25)
	setLiarAndVictim(chain, "G", "C", 25)
	setLiarAndVictim(chain, "H", "A", 25)
	setLiarAndVictim(chain, "H", "B", 25)
	setLiarAndVictim(chain, "H", "C", 25)

	//deleteLatency(chain, "A", "B") //same
	//deleteLatency(chain, "A", "C") //same
	//deleteLatency(chain, "A", "D") //1 less strike, still works
	//deleteLatency(chain, "A", "E") //1 less strike, still works
	//deleteLatency(chain, "A", "F") //1 less strike, still works

	//deleteLatency(chain, "B", "C") //same
	//deleteLatency(chain, "B", "D") //1 less strike, still works
	//deleteLatency(chain, "B", "E") //1 less strike, still works
	//deleteLatency(chain, "B", "F") //1 less strike, still works

	//deleteLatency(chain, "C", "D") //1 less strike, still works
	//deleteLatency(chain, "C", "E") //1 less strike, still works
	//deleteLatency(chain, "C", "F") //1 less strike, still works
	//deleteLatency(chain, "D", "E") //same
	//deleteLatency(chain, "D", "F") //same
	//deleteLatency(chain, "D", "G") // H blacklisted, not G (6 like F, E)
	//deleteLatency(chain, "D", "H") //G blacklisted, not H (6 like F, E)

	//deleteLatency(chain, "E", "F") //same
	//deleteLatency(chain, "E", "G") //H blacklisted, not G (6 like F, D)
	//deleteLatency(chain, "E", "H") //G blacklisted, not H (6 like F, H)
	//deleteLatency(chain, "F", "G") //H blacklisted, not G (6 like D, E)
	deleteLatency(chain, "F", "H") //G blacklisted, not H (6 like D, H)

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
	log.Print(firstBlacklist.ToString())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

}

func TestBlacklistSmallNetworkAssimmetricalLiesSomeUnknown(t *testing.T) {

	// A <-> D does not make sense, not enough info to know who is evil

	N := 4
	d := 1 * time.Nanosecond
	suspicionThreshold := 1

	chain, nodes := simpleChain(N)

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

func TestBlacklistExactlyOneLiarLargeAssimmetricalLiesSomeUnknown(t *testing.T) {

	N := 5
	d := 1 * time.Nanosecond
	suspicionThreshold := 2

	chain, nodes := simpleChain(N)

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
