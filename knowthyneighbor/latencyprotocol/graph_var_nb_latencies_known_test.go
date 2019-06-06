package latencyprotocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBlacklistOneLiarNotAllLatencies(t *testing.T) {
	N := 7
	d := 0 * time.Nanosecond
	withSuspects := true

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := chainWithAllLatenciesSame(N, 10)

	setLiarAndVictim(chain, "N0", "N1", 70)
	setLiarAndVictim(chain, "N0", "N2", 200)
	setLiarAndVictim(chain, "N0", "N4", 20000)
	setLiarAndVictim(chain, "N0", "N5", 200000)
	setLiarAndVictim(chain, "N0", "N6", 2000000)

	for index := range nodeIDs {

		blacklist, err := CreateBlacklist(chain, d, false, false, -1, withSuspects)

		blacklists[index] = blacklist

		require.NoError(t, err)

	}

	firstBlacklist := blacklists[0]

	require.Equal(t, 1, firstBlacklist.Size())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

	//expect both liar and victim to get 2 strikes
	require.Equal(t, 1, strikes[90])
	require.Equal(t, 1, len(strikes))

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}

func TestBlacklistTwoLiarsNotAllLatencies(t *testing.T) {
	N := 14
	d := 0 * time.Nanosecond
	withSuspects := true

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := chainWithAllLatenciesSame(N, 10)

	setLiarAndVictim(chain, "N0", "N1", 70)
	setLiarAndVictim(chain, "N0", "N2", 200)
	setLiarAndVictim(chain, "N0", "N3", 2000)
	setLiarAndVictim(chain, "N0", "N4", 20000)
	setLiarAndVictim(chain, "N0", "N5", 200000)
	setLiarAndVictim(chain, "N0", "N6", 2000000)
	setLiarAndVictim(chain, "N0", "N7", 20000000)
	setLiarAndVictim(chain, "N0", "N8", 200000000)
	setLiarAndVictim(chain, "N0", "N9", 2000000000)
	setLiarAndVictim(chain, "N0", "N10", 20000000000)
	setLiarAndVictim(chain, "N0", "N11", 200000000000)

	setLiarAndVictim(chain, "N1", "N0", 170)
	setLiarAndVictim(chain, "N1", "N2", 1200)
	setLiarAndVictim(chain, "N1", "N3", 12000)
	setLiarAndVictim(chain, "N1", "N4", 120000)
	setLiarAndVictim(chain, "N1", "N5", 1200000)
	setLiarAndVictim(chain, "N1", "N6", 12000000)
	setLiarAndVictim(chain, "N1", "N8", 1200000000)
	setLiarAndVictim(chain, "N1", "N9", 12000000000)
	setLiarAndVictim(chain, "N1", "N10", 120000000000)
	setLiarAndVictim(chain, "N1", "N11", 1200000000000)
	setLiarAndVictim(chain, "N1", "N12", 12000000000000)
	setLiarAndVictim(chain, "N1", "N13", 120000000000000)

	for index := range nodeIDs {

		blacklist, err := CreateBlacklist(chain, d, false, false, -1, withSuspects)

		blacklists[index] = blacklist

		require.NoError(t, err)

	}

	firstBlacklist := blacklists[0]

	require.Equal(t, 2, firstBlacklist.Size())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

	//expect both liar and victim to get 2 strikes
	require.Equal(t, 1, strikes[468])
	require.Equal(t, 1, strikes[462])
	require.Equal(t, 2, len(strikes))

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}
