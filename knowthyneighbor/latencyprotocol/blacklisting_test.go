/*
balcklisting_test tests the creation of blacklists
*/

package latencyprotocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3/log"
)

func TestBlacklist1Liar7Nodes(t *testing.T) {
	N := 7
	d := 0 * time.Nanosecond

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := chainWithAllLatenciesSame(N, 10)

	setLiarAndVictim(chain, "N0", "N1", 70)
	setLiarAndVictim(chain, "N0", "N2", 200)
	setLiarAndVictim(chain, "N0", "N3", 2000)
	setLiarAndVictim(chain, "N0", "N4", 20000)
	setLiarAndVictim(chain, "N0", "N5", 200000)
	setLiarAndVictim(chain, "N0", "N6", 2000000)

	for index := range nodeIDs {

		blacklist, err := CreateBlacklist(chain, d, false, false, -1, true)

		blacklists[index] = blacklist

		require.NoError(t, err)

	}

	firstBlacklist := blacklists[0]

	log.Print(firstBlacklist.ToString())

	require.Equal(t, 1, firstBlacklist.Size())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

	//expect both liar and victim to get 2 strikes
	require.Equal(t, 1, strikes[90])
	require.Equal(t, 1, len(strikes))
	require.True(t, firstBlacklist.ContainsAsString("N0"))

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}

func TestBlacklist2Liars7Nodes(t *testing.T) {
	N := 7
	d := 0 * time.Nanosecond

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := chainWithAllLatenciesSame(N, 10)

	setLiarAndVictim(chain, "N0", "N2", 200)
	setLiarAndVictim(chain, "N0", "N3", 2000)
	setLiarAndVictim(chain, "N0", "N4", 20000)
	setLiarAndVictim(chain, "N0", "N5", 200000)
	setLiarAndVictim(chain, "N0", "N6", 2000000)

	setLiarAndVictim(chain, "N1", "N2", 200)
	setLiarAndVictim(chain, "N1", "N3", 2000)
	setLiarAndVictim(chain, "N1", "N4", 20000)
	setLiarAndVictim(chain, "N1", "N5", 200000)
	setLiarAndVictim(chain, "N1", "N6", 2000000)

	for index := range nodeIDs {

		blacklist, err := CreateBlacklist(chain, d, false, false, -1, true)

		blacklists[index] = blacklist

		require.NoError(t, err)

	}

	firstBlacklist := blacklists[0]

	require.Equal(t, 2, firstBlacklist.Size())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

	log.Print(firstBlacklist.ToString())

	//expect both liar and victim to get 2 strikes
	require.Equal(t, 2, strikes[1])
	require.True(t, firstBlacklist.ContainsAsString("N0"))
	require.True(t, firstBlacklist.ContainsAsString("N1"))

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}

func TestBlacklist2Liars14Nodes(t *testing.T) {
	N := 14
	d := 0 * time.Nanosecond

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
	setLiarAndVictim(chain, "N0", "N12", 2000000000000)
	setLiarAndVictim(chain, "N0", "N13", 20000000000000)

	setLiarAndVictim(chain, "N1", "N0", 170)
	setLiarAndVictim(chain, "N1", "N2", 1200)
	setLiarAndVictim(chain, "N1", "N3", 12000)
	setLiarAndVictim(chain, "N1", "N4", 120000)
	setLiarAndVictim(chain, "N1", "N5", 1200000)
	setLiarAndVictim(chain, "N1", "N6", 12000000)
	setLiarAndVictim(chain, "N1", "N7", 120000000)
	setLiarAndVictim(chain, "N1", "N8", 1200000000)
	setLiarAndVictim(chain, "N1", "N9", 12000000000)
	setLiarAndVictim(chain, "N1", "N10", 120000000000)
	setLiarAndVictim(chain, "N1", "N11", 1200000000000)
	setLiarAndVictim(chain, "N1", "N12", 12000000000000)
	setLiarAndVictim(chain, "N1", "N13", 120000000000000)

	for index := range nodeIDs {

		blacklist, err := CreateBlacklist(chain, d, false, false, -1, true)

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
	require.Equal(t, 2, strikes[468])
	require.Equal(t, 1, len(strikes))

	require.True(t, firstBlacklist.ContainsAsString("N0"))
	require.True(t, firstBlacklist.ContainsAsString("N1"))

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}

func TestBlacklist1Victim(t *testing.T) {
	N := 7
	d := 0 * time.Nanosecond

	blacklists := make([]Blacklistset, N)

	chain, nodeIDs := chainWithAllLatenciesSame(N, 10)

	setLiarAndVictim(chain, "N0", "N6", 2000000)
	setLiarAndVictim(chain, "N1", "N6", 2000000)

	for index := range nodeIDs {

		blacklist, err := CreateBlacklist(chain, d, false, false, -1, true)

		blacklists[index] = blacklist

		require.NoError(t, err)

	}

	firstBlacklist := blacklists[0]

	//none blacklisted - we prefer not blacklisting anyone over blacklisting a victim
	require.Equal(t, 0, firstBlacklist.Size())

	strikes := make(map[int]int, 0)

	for _, strikeNb := range firstBlacklist.Strikes {
		strikes[strikeNb]++
	}

	log.Print(firstBlacklist.ToString())

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklist.Equals(&firstBlacklist))
	}

}
