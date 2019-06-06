/*
 This file allows us to depict the behavior of a network when a liar attempts to infiltrate one or more other clusters at once.

 There are multiple configurable variables (see below)

 Once configured, the test should be run from the terminal within the latencyprotocol folder using the command:

	go test -run TestSoloClusterInfiltrationGraphCreation -timeout=24h


 The generated data can be found under python_graphs/var_clusters, as can the jupyter notebooks to create the graphs

 Note: for increasing number of clusters, execution time of the test grows exponentially
*/
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

**/

func TestSoloClusterInfiltrationGraphCreation(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())

	//create clustered network: reaches max possible strikes at 1978 distance
	//distance := binarySearch(200000, N, clusterSizes)

	//configs ==================================================================================================================
	distance := 1500
	nbClusters := 4
	nbNodesRange := []int{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	withSuspects := true
	//==========================================================================================================================

	filename := "cluster_infiltration_N_" + strconv.Itoa(nbNodesRange[0]) + "_to_" +
		strconv.Itoa(nbNodesRange[len(nbNodesRange)-1]) + "_with_" + strconv.Itoa(nbClusters) + "_clusters"

	if withSuspects {
		filename += "_with_suspects"
	}

	file, err := os.Create("../../python_graphs/var_clusters/data/" + filename + ".csv")
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
				consistentChain, _, _ := chainWithCClusters(len(clusterSize), clusterSize, distance)
				//test

				inconsistentChain := setLiesToClusters(0, consistentChain)

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

func BinarySearch(initDistance int, N int, clusterSizes []int, withSuspects bool) int {

	low := 0
	high := initDistance

	median := (low + high) / 2

	for low <= high {
		median = (low + high) / 2

		//create clustered network: reaches max possible strikes at 1978 distance
		consistentChain, _, _ := chainWithCClusters(len(clusterSizes), clusterSizes, median)

		//set liar
		inconsistentChain := setLiesToClusters(0, consistentChain)

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
