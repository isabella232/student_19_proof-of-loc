package latencyprotocol

import (
	"math/rand"
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

func Test_simulate_multicluster_infiltration(t *testing.T) {

	N := 10

	clusterSizes := []int{4, 3, 3}

	//create clustered network: reaches max possible strikes at 1978 distance
	consistentChain, _, _ := create_graph_with_C_clusters(3, clusterSizes, 1978)

	//set liar
	inconsistentChain := set_lies_to_clusters(0, consistentChain)

	thresh := UpperThreshold(N)
	//threshold := strconv.Itoa(thresh)
	strikes, _ := CreateBlacklist(inconsistentChain, 0, false, true, 0)
	blacklist, err := CreateBlacklist(inconsistentChain, 0, false, true, thresh)
	if err != nil {
		log.Print(err)
	}

	log.Print(thresh)
	log.Print(strikes.ToString())
	log.Print(blacklist.IsEmpty())
}

func create_graph_with_C_clusters(C int, clusterSizes []int, distance int) (*Chain, []*Chain, [][]string) {

	clusters := make([]*Chain, C)
	nodeLists := make([][]string, C)

	N := 0

	for i := 0; i < C; i++ {
		cluster, nodeList := consistentChain(clusterSizes[i], N)
		clusters[i] = cluster
		nodeLists[i] = nodeList
		N += clusterSizes[i]
	}

	masterBlocks := make([]*Block, N)
	masterIndex := 0

	//Steps:
	//1: for each chain, add latency to blocks of all other chains (careful: latencies go 2 ways -> only give forward)
	//2: Connect all chains

	//1
	//for each chain
	for j := 0; j < C; j++ {
		cluster := clusters[j]

		//for each block in chain
		for b := 0; b < len(cluster.Blocks); b++ {
			block := cluster.Blocks[b]
			blockId := j + b

			//for each other chain
			for nl := j + 1; nl < C; nl++ {
				nodes := nodeLists[nl]
				for n := 0; n < len(nodes); n++ {
					node := nodes[n]
					newLat := ConfirmedLatency{time.Duration(distance), nil, time.Now(), nil}
					block.Latencies[node] = newLat
					clusters[nl].Blocks[n].Latencies[numbersToNodes(blockId)] = newLat
				}

			}

			//Copy completed block to big graph
			masterBlocks[masterIndex] = block
			masterIndex++

		}
	}

	//2

	masterChain := Chain{masterBlocks, []byte("testBucketName")}

	return &masterChain, clusters, nodeLists

}

func set_lies_to_clusters(liarID int, consistentChain *Chain) *Chain {

	inconsistentChain := consistentChain.Copy()

	for i := 0; i < len(inconsistentChain.Blocks); i++ {
		block := inconsistentChain.Blocks[i]

		_, isPresent := block.Latencies[numbersToNodes(liarID)]
		if isPresent {

			//Normal range within cluster
			lat := rand.Intn(500) + 500
			inconsistentChain.Blocks[liarID].Latencies[numbersToNodes(i)] = ConfirmedLatency{time.Duration(lat), nil, time.Now(), nil}
			block.Latencies[numbersToNodes(liarID)] = ConfirmedLatency{time.Duration(lat), nil, time.Now(), nil}
		}
	}
	return inconsistentChain
}
