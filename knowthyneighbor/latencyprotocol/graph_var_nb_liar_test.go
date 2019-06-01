package latencyprotocol

/**
Let’s fix the coordinates of the nodes // done by default by graph generation's pseudo-randomness
Let’s fix the lies. //done by pseudorandom generation of function getLies
But: we vary the set of liars.
Let’s have 100 such sets, resulting in 100 different result sets.
Two sets of experiments: one where liar sets are random, one where liers within a set are close together
*/

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"go.dedis.ch/onet/v3/log"
)

func TestFixedLieIncreasingNbLiarsCreation(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())

	lowerBoundLies := 1000
	upperBoundLies := 50000

	nbNodes := 100
	nbLiars := 33
	nbLiarCombinations := 100
	withSuspects := true

	graphDesign := &GraphDesign{nbNodes, nbLiars, nbNodes, 500, 1000, lowerBoundLies, upperBoundLies, nbLiarCombinations}

	err := CreateFixedLieIncreasingLiesData("test_map_fixed_lies_x_liars_clustered_100_with_suspects", false, graphDesign, withSuspects)
	if err != nil {
		log.Print(err)
	}

}

func CreateFixedLieIncreasingLiesData(filename string, randomLiars bool, graphDesign *GraphDesign, withSuspects bool) error {

	//4) Create a graph where each original latency is on the x-axis,
	//each corresponding latency actually recorded in the chain is on the y-axis,
	//and if the nodes at the ends of the latency (x,y) are in the blacklist, give it a different color.
	// 0, 1 or 2 nodes recorded as blacklisted
	//=> configure X, Y, Blacklist values for graphing, write to file

	N := graphDesign.NbNodes

	//1) Create chain with No TIVs or liars
	consistentChain, _ := consistentChain(N, 0)
	log.Print("Created Consistent Graph")

	var liarSets [][]int

	if randomLiars {
		liarSets = Get_M_subsets_of_K_liars_out_of_N_nodes(graphDesign.NbLiarCombinations, graphDesign.NbLiars, graphDesign.NbNodes)
	} else {
		log.Print("Picking clustered liars")
		liarSets = pickClusteredLiars(consistentChain, graphDesign.NbLiars, graphDesign.NbLiarCombinations)
	}

	file, err := os.Create("../../python_graphs/data/fixed_lies/" + filename + ".csv")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	fmt.Fprintln(file, "node,is_liar,is_blacklisted,lie,lieClusterSize")

	log.Print("Getting lies")
	lies := GetLies(graphDesign.NbLiars, graphDesign.NbVictims, graphDesign.LowerBoundLies, graphDesign.UpperBoundLies)

	log.Print("got lies")

	lieClusterSizes := []int{5, 10, 20, 33}

	for index, liarSet := range liarSets {

		log.Print(strconv.Itoa(index))

		for _, lieClusterSize := range lieClusterSizes {
			subset := liarSet[:lieClusterSize]

			_, unthreshedBlacklist, mapping, err := createLyingNetworkWithMapping(&subset, graphDesign, consistentChain, &lies, withSuspects)
			if err != nil {
				return err
			}
			thresh := UpperThreshold(N)
			//threshold := strconv.Itoa(thresh)

			blacklist := unthreshedBlacklist.GetBlacklistWithThreshold(thresh)

			if err != nil {
				return err
			}

			for i := 0; i < N; i++ {
				nodei := numbersToNodes(i)
				isLiar := ContainsInt(liarSet, i)
				isBlacklisted := blacklist.ContainsAsString(nodei)
				if isLiar {
					for _, lie := range mapping[i] {
						fmt.Fprintln(
							file, nodei+","+strconv.FormatBool(isLiar)+","+strconv.FormatBool(isBlacklisted)+","+strconv.Itoa(lie)+","+strconv.Itoa(lieClusterSize))
					}
				} else {
					fmt.Fprintln(
						file, nodei+","+strconv.FormatBool(isLiar)+","+strconv.FormatBool(isBlacklisted)+","+strconv.Itoa(0)+","+strconv.Itoa(lieClusterSize))
				}

			}
		}

	}

	//5) Repeat 1-4 for a new chain with a different number of nodes (edited)
	return nil

}
