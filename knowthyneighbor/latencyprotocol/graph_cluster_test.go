package latencyprotocol

import (
	"fmt"
	"math"
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

func Test_simulate_multicluster_infiltration(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())

	//create clustered network: reaches max possible strikes at 1978 distance
	//distance := binarySearch(200000, N, clusterSizes)

	distance := 100000

	nbClusters := 2

	withSuspects := true

	file, err := os.Create("../../python_graphs/data/clusters/low_thresh_size_imbalance_" + strconv.Itoa(nbClusters) + "_with_suspects.csv")
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

				inconsistentChain := Set_lies_to_clusters(0, consistentChain)

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

func RotatedArrays(array []int) [][]int {
	nbElems := len(array)
	arrays := make([][]int, nbElems)
	arrays[0] = array

	for i := 1; i < nbElems; i++ {
		array2 := make([]int, nbElems)
		for j := 0; j < nbElems; j++ {
			array2[j] = array[(i+j)%nbElems]
		}
		arrays[i] = array2
	}

	return arrays
}

func Create_graph_with_C_clusters(C int, clusterSizes []int, distance int) (*Chain, []*Chain, [][]string) {

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

			//for each other chain
			for nl := 0; nl < C; nl++ {
				if nl != j {
					nodes := nodeLists[nl]
					for n := 0; n < len(nodes); n++ {
						node := nodes[n]
						newLat := ConfirmedLatency{time.Duration(distance), nil, time.Now(), nil}
						block.Latencies[node] = newLat
						clusters[nl].Blocks[n].Latencies[numbersToNodes(masterIndex)] = newLat
					}

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

func Set_lies_to_clusters(liarID int, consistentChain *Chain) *Chain {

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

func BinarySearch(initDistance int, N int, clusterSizes []int, withSuspects bool) int {

	low := 0
	high := initDistance

	median := (low + high) / 2

	for low <= high {
		median = (low + high) / 2

		//create clustered network: reaches max possible strikes at 1978 distance
		consistentChain, _, _ := Create_graph_with_C_clusters(len(clusterSizes), clusterSizes, median)

		//set liar
		inconsistentChain := Set_lies_to_clusters(0, consistentChain)

		thresh := UpperThreshold(N)
		//threshold := strconv.Itoa(thresh)
		blacklist, err := CreateBlacklist(inconsistentChain, 0, false, true, thresh, withSuspects)
		if err != nil {
			log.Print(err)
		}

		if blacklist.IsEmpty() {
			low = median + 1
		} else {
			high = median - 1
		}
	}

	return median
}

func TestSubsets(t *testing.T) {
	sols := SubsetLengthKSummingToS(3, 11)

	for _, sol := range sols {
		log.Print(sol)
	}

}

func SubsetLengthKSummingToS(K int, S int) [][]int {

	original := make([]int, S-1)
	for i := 1; i < S; i++ {
		original[i-1] = i
	}
	powerSet := PowerSet(original)

	subset := make([][]int, 0)

	for _, set := range powerSet {
		if len(set) == K {
			if CheckSum(set, S) {
				subset = append(subset, set)
			}

			vars := VariantsOfSet(set, 0, 0, make([][]int, 0))
			for _, variant := range vars {
				if CheckSum(variant, S) {
					subset = append(subset, variant)
				}
			}
		}
	}

	return subset
}

func CopyArray(array []int) []int {
	newArray := make([]int, len(array))
	for i, val := range array {
		newArray[i] = val
	}
	return newArray
}

func CheckSum(set []int, sum int) bool {
	setSum := 0
	for _, elem := range set {
		setSum += elem
	}
	return sum == setSum
}

func PowerSet(original []int) [][]int {
	powerSetSize := int(math.Pow(2, float64(len(original))))
	result := make([][]int, 0, powerSetSize)

	var index int
	for index < powerSetSize {
		var subSet []int

		for j, elem := range original {
			if index&(1<<uint(j)) > 0 {
				subSet = append(subSet, elem)
			}
		}
		result = append(result, subSet)
		index++
	}
	return result
}

func VariantsOfSet(original []int, valueIndex int, startIndex int, result [][]int) [][]int {
	if startIndex >= len(original) || valueIndex >= len(original) {
		return result
	}
	newArray := CopyArray(original)
	newArray[startIndex] = original[valueIndex]
	result = append(result, newArray)

	result = VariantsOfSet(original, valueIndex, startIndex+1, result)
	result = VariantsOfSet(original, valueIndex+1, startIndex, result)
	result = VariantsOfSet(newArray, valueIndex, startIndex+1, result)
	result = VariantsOfSet(newArray, valueIndex+1, startIndex, result)

	return result

}
