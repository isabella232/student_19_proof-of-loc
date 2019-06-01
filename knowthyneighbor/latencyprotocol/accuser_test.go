/*
graph_accuser_test allows us to test a complementary method of identifying liars:
if a nodes is acting suspiciously (if it has more strikes than average, but not enough to be blacklisted),
we run the following steps:
	1) for each triangle with this node as an edge, we add a strike to the involved nodes if triangle inequality is violated
	2) for each node, we check the number of strikes. If it is at most N/3, we keep the node
	(A non-liar can have at most floor(N/3) strikes, one strike gotten from each liar.)
	3) We count the number of kept nodes. If they make up more than 2 thirds (2N/3) of the network, the node is a liar because
	it has more violations than could be caused by a separate group of liars.

*/

package latencyprotocol

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3/log"
)

/**
Experiments on a node trying to make itself closer to multiple clusters of nodes. Take 2 clusters.
The size of the clusters, the distance between the clusters.
Keep the cluster sizes the same, vary the distance between them
Keep the distance the same, keep the sum of nodes in clusters the same, vary the sizes of these clusters
Output: is the node detected or not

Case 1:
**/

func Test_accuser_analysis(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())

	//create clustered network: reaches max possible strikes at 1978 distance
	//distance := binarySearch(200000, N, clusterSizes)

	distance := 100000

	N := 10
	//nbClusters := 2
	clusterSizes := []int{5, 5}

	consistentChain, _, _ := Create_graph_with_C_clusters(len(clusterSizes), clusterSizes, distance)
	//test
	inconsistentChain := Set_lies_to_clusters(0, consistentChain)

	thresh := UpperThreshold(N)
	//threshold := strconv.Itoa(thresh)
	basicBlacklist, err := CreateBlacklist(inconsistentChain, 0, false, true, thresh, false)
	if err != nil {
		log.Print(err)
	}

	extendedBlacklist, err := CreateBlacklist(inconsistentChain, 0, false, true, thresh, true)
	if err != nil {
		log.Print(err)
	}

	require.Equal(t, 1, extendedBlacklist.Size())

	for suspect := range extendedBlacklist.Strikes {
		probablyLiar := SuspectIsLiar(inconsistentChain, suspect, N)
		if probablyLiar {
			log.Print("Probably liar: " + suspect)
			//require.Equal(t, suspect, "N0")
		} else {
			log.Print("Probably victim: " + suspect)
			//require.Fail(t, "should not detect victims")
		}
	}

	require.Empty(t, basicBlacklist.Strikes)

}
