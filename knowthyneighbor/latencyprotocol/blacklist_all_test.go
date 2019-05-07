package latencyprotocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3/log"
	sigAlg "golang.org/x/crypto/ed25519"
)

func TestBlacklistOneLiarOneVictim(t *testing.T) {
	N := 4
	d := 1 * time.Nanosecond

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N0", "N3", 25)

	for index := range nodeIDs {

		blacklist, err := CreateBlacklist(chain, d)

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

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N0", "N1", 25)
	setLiarAndVictim(chain, "N0", "N2", 25)

	for index := range nodeIDs {

		blacklist, err := CreateBlacklist(chain, d)

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

func TestBlacklistFiveNodesOneLiarTwoVictims(t *testing.T) {
	N := 5
	d := 1 * time.Nanosecond

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N4", "N0", 25)
	setLiarAndVictim(chain, "N4", "N1", 25)

	for index := range nodeIDs {

		blacklist, err := CreateBlacklist(chain, d)

		blacklists[index] = blacklist

		require.NoError(t, err)

	}

	firstBlacklist := blacklists[1]

	require.Equal(t, N, firstBlacklist.Size())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

	require.Equal(t, 1, strikes[4])
	require.Equal(t, 4, strikes[2])

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}

func TestBlacklistEightNodesTwoLiarsThreeVictims(t *testing.T) {
	N := 8
	d := 1 * time.Nanosecond
	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N6", "N0", 25)
	setLiarAndVictim(chain, "N6", "N1", 25)
	setLiarAndVictim(chain, "N6", "N2", 25)
	setLiarAndVictim(chain, "N7", "N0", 25)
	setLiarAndVictim(chain, "N7", "N1", 25)
	setLiarAndVictim(chain, "N7", "N2", 25)

	for index := range nodeIDs {

		blacklist, err := CreateBlacklist(chain, d)

		blacklists[index] = blacklist

		require.NoError(t, err)

	}

	firstBlacklist := blacklists[0]

	print(firstBlacklist.ToString())

	require.Equal(t, N, firstBlacklist.Size())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

	//expect liars to get nine strikes and the rest to get 6
	require.Equal(t, 6, strikes[6])
	require.Equal(t, 2, strikes[9])

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}

func TestBlacklistSmallNetworkAssimmetricalLies(t *testing.T) {

	// A <-> D does not make sense, not enough info to know who is evil

	N := 4
	d := 1 * time.Nanosecond
	chain, nodes := simpleChain(N)

	chain.Blocks[0].Latencies["N3"] = ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[3].Latencies["N0"] = ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[1].Latencies["N3"] = ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[3].Latencies["N1"] = ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil}

	blacklists := make([]Blacklistset, N)

	for index := range nodes {

		blacklist, err := CreateBlacklist(chain, d)

		require.NoError(t, err)
		log.Print(blacklist.ToString())
		blacklists[index] = blacklist

	}

	firstBlacklist := blacklists[0]

	require.Equal(t, 2, firstBlacklist.Size())
	require.True(t, firstBlacklist.Contains(sigAlg.PublicKey([]byte("N0")), 0))
	require.True(t, firstBlacklist.Contains(sigAlg.PublicKey([]byte("N3")), 0))

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}

func TestBlacklistExactlyOneLiarLargeAssimmetricalLies(t *testing.T) {

	N := 5
	d := 1 * time.Nanosecond
	suspicionThreshold := UpperThreshold(N)

	log.Print(suspicionThreshold)

	chain, nodes := simpleChain(N)

	chain.Blocks[1].Latencies["N2"] = ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[1].Latencies["N3"] = ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil}

	chain.Blocks[2].Latencies["N1"] = ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[2].Latencies["N3"] = ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[2].Latencies["N4"] = ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil}

	chain.Blocks[3].Latencies["N1"] = ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[3].Latencies["N2"] = ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[3].Latencies["N4"] = ConfirmedLatency{time.Duration(20 * time.Nanosecond), nil, time.Now(), nil}

	chain.Blocks[4].Latencies["N2"] = ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[4].Latencies["N3"] = ConfirmedLatency{time.Duration(20 * time.Nanosecond), nil, time.Now(), nil}

	blacklists := make([]Blacklistset, N)

	for index := range nodes {

		blacklist, err := CreateBlacklist(chain, d)

		require.NoError(t, err)
		log.Print(blacklist.ToString())
		//require.NotZero(t, blacklist.Size())
		blacklists[index] = blacklist

	}

	firstBlacklist := blacklists[0]

	log.Print(firstBlacklist.ToString())
	log.Print(firstBlacklist.Contains(sigAlg.PublicKey([]byte("N2")), suspicionThreshold))

	require.Equal(t, 1, firstBlacklist.Size())
	require.True(t, firstBlacklist.Contains(sigAlg.PublicKey([]byte("N2")), suspicionThreshold))

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}

func TestMaxTriangles(t *testing.T) {
	N := 26
	d := 1 * time.Nanosecond

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N3", "N0", 25)
	setLiarAndVictim(chain, "N3", "N1", 25)
	setLiarAndVictim(chain, "N3", "N2", 25)
	setLiarAndVictim(chain, "N3", "N4", 25)
	setLiarAndVictim(chain, "N3", "N5", 25)
	setLiarAndVictim(chain, "N3", "N6", 25)
	setLiarAndVictim(chain, "N3", "N7", 25)
	setLiarAndVictim(chain, "N3", "N8", 25)

	for index := range nodeIDs {

		blacklist, err := CreateBlacklist(chain, d)

		blacklists[index] = blacklist

		require.NoError(t, err)

	}

	firstBlacklist := blacklists[1]

	log.Print(firstBlacklist.ToString())

}
