package latencyprotocol

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

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

func Test_simulate_multicluster_multiliar_infiltration(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())

	//create clustered network: reaches max possible strikes at 1978 distance
	//distance := binarySearch(200000, N, clusterSizes)

	distance := 100000

	nbClusters := 2
	nbLiars := 2

	withSuspects := true

	file, err := os.Create("../../python_graphs/data/clusters/" +
		strconv.Itoa(nbLiars) + "_liar_size_imbalance_" + strconv.Itoa(nbClusters) + "_with_suspects.csv")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	columns := "N,"

	for i := 1; i <= nbClusters; i++ {
		columns += "c" + strconv.Itoa(i) + ","
	}
	columns += "liar_caught"
	fmt.Fprintln(file, columns)

	for N := 11; N < 26; N++ {
		log.Print(N)
		clusterSizes := SubsetLengthKSummingToS(nbClusters, N)
		for _, clusterSizeOrig := range clusterSizes {
			clusterSizeRotated := RotatedArrays(clusterSizeOrig)
			for _, clusterSize := range clusterSizeRotated {
				consistentChain, _, _ := Create_graph_with_C_clusters(len(clusterSize), clusterSize, distance)
				//test

				inconsistentChain := set_multiple_lies_to_clusters(2, consistentChain)

				thresh := UpperThreshold(N)
				//threshold := strconv.Itoa(thresh)
				blacklist, err := CreateBlacklist(inconsistentChain, 0, false, true, thresh, withSuspects)
				//unthresholded, err := CreateBlacklist(inconsistentChain, 0, false, true, 0)
				if err != nil {
					log.Print(err)
				}

				line := strconv.Itoa(N) + ","

				for i := 0; i < nbClusters; i++ {
					line += strconv.Itoa(clusterSize[i]) + ","
				}
				line += strconv.FormatBool(!blacklist.IsEmpty())

				fmt.Fprintln(file, line)
			}
		}

	}

}

func set_multiple_lies_to_clusters(nbLiars int, consistentChain *Chain) *Chain {

	inconsistentChain := consistentChain.Copy()

	for i := 0; i < len(inconsistentChain.Blocks); i++ {
		block := inconsistentChain.Blocks[i]

		for liarID := 0; liarID < nbLiars; liarID++ {

			_, isPresent := block.Latencies[numbersToNodes(liarID)]
			if isPresent {

				//Normal range within cluster
				lat := rand.Intn(500) + 500
				inconsistentChain.Blocks[liarID].Latencies[numbersToNodes(i)] = ConfirmedLatency{time.Duration(lat), nil, time.Now(), nil}
				block.Latencies[numbersToNodes(liarID)] = ConfirmedLatency{time.Duration(lat), nil, time.Now(), nil}
			}
		}
	}

	return inconsistentChain
}
