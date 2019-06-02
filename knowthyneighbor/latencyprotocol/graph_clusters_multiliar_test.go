/*
 This file allows us to depict the behavior of a network when multiple liars attempt to infiltrate one or more other clusters at once.

 There are multiple configurable variables (see below)

 Once configured, the test should be run from the terminal within the latencyprotocol folder using the command:

	go test -run TestMultiliarClusterInfiltrationGraphCreation -timeout=24h


 The generated data can be found under python_graphs/var_clusters, as can the jupyter notebooks to create the graphs
*/
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

func TestMultiliarClusterInfiltrationGraphCreation(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())

	//configs ==================================================================================================================
	distance := 100000
	nbClusters := 2
	nbLiars := 4
	nbNodesRange := []int{13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26}
	withSuspects := true
	//==========================================================================================================================

	filename := "cluster_infiltration_N_" + strconv.Itoa(nbNodesRange[0]) + "_to_" +
		strconv.Itoa(nbNodesRange[len(nbNodesRange)-1]) + "_with_" + strconv.Itoa(nbClusters) + "_clusters_" + strconv.Itoa(nbLiars) + "_liars"

	if withSuspects {
		filename += "_with_suspects"
	}

	file, err := os.Create("../../python_graphs/var_clusters_multiliar/data/" + filename + ".csv")
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

	for _, N := range nbNodesRange {
		log.Print(N)
		clusterSizes := SubsetLengthKSummingToS(nbClusters, N)
		for _, clusterSizeOrig := range clusterSizes {
			clusterSizeRotated := RotatedArrays(clusterSizeOrig)
			for _, clusterSize := range clusterSizeRotated {
				consistentChain, _, _ := Create_graph_with_C_clusters(len(clusterSize), clusterSize, distance)

				inconsistentChain := set_multiple_lies_to_clusters(nbLiars, consistentChain)

				thresh := UpperThreshold(N)
				blacklist, err := CreateBlacklist(inconsistentChain, 0, false, true, thresh, withSuspects)
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
