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

type GraphDesign struct {
	NbNodes         int
	NbLiars         int
	NbVictims       int
	LowerBoundTruth int
	UpperBoundTruth int
	LowerBoundLies  int
	UpperBoundLies  int
}

func TestFixedLieRandomLiarGraphCreation(t *testing.T) {

	lowerBoundLies := 1000
	upperBoundLies := 100000

	nbNodes := 100
	nbLiars := 33

	nbLiarCombinations := 100

	graphDesign := &GraphDesign{nbNodes, nbLiars, nbNodes, 500, 1000, lowerBoundLies, upperBoundLies}

	liarSets := Get_M_subsets_of_K_liars_out_of_N_nodes(nbLiarCombinations, nbLiars, nbNodes)

	err := CreateFixedLiePercentageGraphData("test_100_Nodes_33_liars_fixed_lies_count_liars_bl", true, liarSets, graphDesign)
	if err != nil {
		log.Print(err)
	}

}

func CreateFixedLiePercentageGraphData(filename string, randomLiars bool, liarSets [][]int, graphDesign *GraphDesign) error {

	//4) Create a graph where each original latency is on the x-axis,
	//each corresponding latency actually recorded in the chain is on the y-axis,
	//and if the nodes at the ends of the latency (x,y) are in the blacklist, give it a different color.
	// 0, 1 or 2 nodes recorded as blacklisted
	//=> configure X, Y, Blacklist values for graphing, write to file

	N := graphDesign.NbNodes

	//1) Create chain with No TIVs or liars
	consistentChain, _ := consistentChain(N)
	log.Print("Created Consistent Graph")

	/*testBlacklist, _ := CreateBlacklist(consistentChain, 0, false, true, 0)

	log.Print("Created Blacklist for consistent")

	if !testBlacklist.IsEmpty() {
		log.Print(testBlacklist.ToString())
		return errors.New("Original graph has triangle inequality violations")
	}*/

	file, err := os.Create("../../python_graphs/data/fixed_lies/" + filename + ".csv")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	fmt.Fprintln(file, "node,is_liar,is_blacklisted")

	lies := GetLies(graphDesign.NbLiars, graphDesign.NbVictims, graphDesign.LowerBoundLies, graphDesign.UpperBoundLies)

	for _, liarSet := range liarSets {

		_, unthreshedBlacklist := createLyingNetwork(&liarSet, graphDesign, consistentChain, &lies)
		thresh := UpperThreshold(N)
		//threshold := strconv.Itoa(thresh)

		blacklist := unthreshedBlacklist.GetBlacklistWithThreshold(thresh)

		if err != nil {
			return err
		}

		for i := 0; i < N; i++ {
			nodei := numbersToNodes(i)
			isLiar := containsNodeWithID(liarSet, i)
			isBlacklisted := blacklist.ContainsAsString(nodei)

			fmt.Fprintln(file, nodei+","+strconv.FormatBool(isLiar)+","+strconv.FormatBool(isBlacklisted))

		}

	}

	//5) Repeat 1-4 for a new chain with a different number of nodes (edited)
	return nil

}

func containsNodeWithID(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func createLyingNetwork(liarSet *([]int), graphDesign *GraphDesign, consistentChain *Chain, lies *([]int)) (*Chain, *Blacklistset) {

	N := graphDesign.NbNodes

	//2) Modify some of the latencies so they might no longer be consistent
	inconsistentChain := consistentChain.Copy()
	log.Print("Copied Consistent Graph")

	//All liars target 1 victim
	/*victim := nbLiars
	for n1 := range liarSet {

		oldLatency := int(consistentChain.Blocks[n1].Latencies[numbersToNodes(victim)].Latency.Nanoseconds())

		lie := lies[n1+victim*N]

		setLiarAndVictim(inconsistentChain, numbersToNodes(n1), numbersToNodes(victim), time.Duration(lie + oldLatency))

	}*/

	//println("Size lies: " + strconv.Itoa(len(*lies)))
	//println("Size liar set: " + strconv.Itoa(len(*liarSet)))
	for n1Index, n1 := range *liarSet {

		//liars not attacking each other: n2 := nbLiars
		for n2 := 0; n2 < N; n2++ {
			if n1 != n2 {

				/*println("n1: " + strconv.Itoa(n1))
				println("n2: " + strconv.Itoa(n2))
				println("index: " + strconv.Itoa(n2+n1Index*N-1))*/

				lie := (*lies)[n2+n1Index*N-1]
				oldLatency := int(consistentChain.Blocks[n1].Latencies[numbersToNodes(n2)].Latency.Nanoseconds())

				setLiarAndVictim(inconsistentChain, numbersToNodes(n1), numbersToNodes(n2), time.Duration(oldLatency+lie))
			}
		}
	}

	log.Print("Lies set")

	//3) Create the blacklist of the chain
	blacklist, _ := CreateBlacklist(inconsistentChain, 0, false, true, 0)

	log.Print("Create blacklist")

	return inconsistentChain, &blacklist
}

// Get_M_subsets_of_K_liars_out_of_N_nodes does exactly what the name says with numbers as nodes
func Get_M_subsets_of_K_liars_out_of_N_nodes(M int, K int, N int) [][]int {
	superSet := makeRange(N)
	current := make([]int, 0)
	solution := make([][]int, 0)
	solution = getSubsets(superSet, K, 0, current, solution, M)

	return solution[:M]
}

//GetLies generates a pseudo-random set of lies to be reused during fixed lie testing
func GetLies(nbLiars int, nbVictims int, lowerBound int, upperBound int) []int {
	lies := make([]int, nbLiars*nbVictims)
	for i := 0; i < nbLiars; i++ {
		for j := 0; j < nbVictims; j++ {
			lat := rand.Intn(upperBound-lowerBound) + lowerBound
			lies[j+i*nbVictims] = lat
		}
	}
	return lies
}
