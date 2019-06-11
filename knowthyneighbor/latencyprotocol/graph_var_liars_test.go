/*
 This file allows us to measure how many liars are blacklisted, depending on the choice of liars for a fixed set of lies.

 There are multiple configurable variables (see below)

 Once configured, the test should be run from the terminal within the latencyprotocol folder using the command:

	go test -run TestVarLiarsGraphCreation -timeout=24h


 The generated data can be found under python_graphs/var_liars, as can the jupyter notebooks to create the graphs
*/
package latencyprotocol

/**
Let’s fix the coordinates of the nodes // done by default by graph generation's pseudo-randomness
Let’s fix the lies. //done by pseudorandom generation of function getLies
But: we vary the set of liars.
Let’s have 100 such sets, resulting in 100 different result sets.
Two sets of experiments: one where liar sets are random, one where liers within a set are close together
*/

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"go.dedis.ch/onet/v3/log"
)

type GraphDesign struct {
	NbNodes            int
	NbLiars            int
	NbVictims          int
	LowerBoundTruth    int
	UpperBoundTruth    int
	LowerBoundLies     int
	UpperBoundLies     int
	NbLiarCombinations int
}

func TestVarLiarsGraphCreation(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())

	//configs =====================================================================================================
	lowerBoundLies := 1000 //lower bound on difference between true latency and lie told about it
	upperBoundLies := 5000 //upper bound on difference between true latency and lie told about it
	nbNodes := 100
	nbLiars := 33
	nbLiarCombinations := 100 //nb different combinations of liars chosen throughout test
	randomLiars := true       //whether the liars are chosen randomly or within same cluster
	withSuspects := true      //activate enhanced blacklisting
	//=============================================================================================================

	if int(nbNodes/3) < nbLiars {
		log.Print("Error: cannot have more than N/3 liars")
		return
	}

	random := "random_liars"
	if !randomLiars {
		random = "clustered_liars"
	}

	filename := "test_" +
		strconv.Itoa(nbNodes) + "_nodes_" +
		strconv.Itoa(nbLiars) + "_liars" +
		"_var_liars_distance_" + strconv.Itoa(upperBoundLies) +
		"_" + random +
		"_" + strconv.Itoa(nbLiarCombinations) + "_combinations"

	if withSuspects {
		filename += "_with_suspects"
	}

	graphDesign := &GraphDesign{nbNodes, nbLiars, nbNodes, 500, lowerBoundLies, lowerBoundLies, upperBoundLies, nbLiarCombinations}

	err := CreateFixedLieToEffectMap(filename, randomLiars, graphDesign, withSuspects)
	if err != nil {
		log.Print(err)
	}

}

func CreateFixedLieToEffectMap(filename string, randomLiars bool, graphDesign *GraphDesign, withSuspects bool) error {

	// Create a graph where each original latency is on the x-axis,
	//each corresponding latency actually recorded in the chain is on the y-axis,
	//and if the nodes at the ends of the latency (x,y) are in the blacklist, give it a different color.
	// 0, 1 or 2 nodes recorded as blacklisted
	//=> configure X, Y, Blacklist values for graphing, write to file

	N := graphDesign.NbNodes

	//1) Create chain with No TIVs or liars
	consistentChain, _ := chainWithOnlyConsistentLatencies(N, 0)
	log.Print("Created Consistent Graph")

	var liarSets [][]int

	if randomLiars {
		liarSets = GetMSubsetsOfKLiarsOutOfNNodes(graphDesign.NbLiarCombinations, graphDesign.NbLiars, graphDesign.NbNodes)
	} else {
		log.Print("Picking clustered liars")
		liarSets = pickClusteredLiars(consistentChain, graphDesign.NbLiars, graphDesign.NbLiarCombinations)
	}

	file, err := os.Create("../../python_graphs/var_liars/data/" + filename + ".csv")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	fmt.Fprintln(file, "node,is_liar,is_blacklisted,lie,cluster")

	log.Print("Getting lies")
	lies := GetLies(graphDesign.NbLiars, graphDesign.NbVictims, graphDesign.LowerBoundLies, graphDesign.UpperBoundLies)

	log.Print("got lies")
	for index, liarSet := range liarSets {

		log.Print(strconv.Itoa(index))

		_, blacklist, mapping, err := createLyingNetworkWithMapping(&liarSet, graphDesign, consistentChain, &lies, withSuspects)
		if err != nil {
			return err
		}

		log.Print("Blacklist size: " + strconv.Itoa(blacklist.Size()))

		if withSuspects {

		}

		if err != nil {
			return err
		}

		for i := 0; i < N; i++ {
			nodei := numbersToNodes(i)
			isLiar := ContainsInt(liarSet, i)
			isBlacklisted := blacklist.ContainsAsString(nodei)
			if isLiar {
				for _, lie := range mapping[i] {
					fmt.Fprintln(file, nodei+","+strconv.FormatBool(isLiar)+","+strconv.FormatBool(isBlacklisted)+","+strconv.Itoa(lie)+","+strconv.Itoa(index))
				}
			} else {
				fmt.Fprintln(file, nodei+","+strconv.FormatBool(isLiar)+","+strconv.FormatBool(isBlacklisted)+","+strconv.Itoa(0)+","+strconv.Itoa(index))
			}

		}

	}

	return nil

}

func createLyingNetworkWithMapping(
	liarSet *([]int), graphDesign *GraphDesign,
	consistentChain *Chain, lies *([]int), withSuspects bool) (*Chain, *Blacklistset, map[int][]int, error) {

	N := graphDesign.NbNodes
	nodeLieMap := make(map[int][]int)

	//2) Modify some of the latencies so they might no longer be consistent
	inconsistentChain := consistentChain.Copy()
	log.Print("Copied Consistent Graph")

	takenLies := make(map[int]bool)
	nbLies := len(*lies)

	log.Print("nbLies: " + strconv.Itoa(nbLies))
	for i := 0; i < nbLies; i++ {
		takenLies[i] = false
	}

	log.Print("size liar set: " + strconv.Itoa(len(*liarSet)))

	sort.Ints(*liarSet)

	liesSet := 0

	for _, n1 := range *liarSet {
		nodeLieMap[n1] = make([]int, 0)

		for n2 := 0; n2 < N; n2++ {

			if n1 != n2 && !(ContainsInt(*liarSet, n2) && n2 <= n1) {

				lieIndex := rand.Intn(N)

				initLie := lieIndex

				for takenLies[lieIndex] == true {
					lieIndex = (lieIndex + 1) % nbLies

					if lieIndex == initLie {
						log.Print("no more lies left")
						log.Print(nbLies)
						log.Print(liesSet)
						return nil, nil, nil, errors.New("Not enough lies")
					}
				}

				takenLies[lieIndex] = true

				lie := (*lies)[lieIndex]
				oldLatency := int(consistentChain.Blocks[n1].Latencies[numbersToNodes(n2)].Latency.Nanoseconds())

				setLiarAndVictim(inconsistentChain, numbersToNodes(n1), numbersToNodes(n2), time.Duration(oldLatency+lie))
				nodeLieMap[n1] = append(nodeLieMap[n1], lie)
				if ContainsInt(*liarSet, n2) {
					if nodeLieMap[n2] != nil {
						nodeLieMap[n2] = append(nodeLieMap[n2], lie)
					} else {
						nodeLieMap[n2] = make([]int, 0)
						nodeLieMap[n2] = append(nodeLieMap[n2], lie)
					}

				}
				liesSet++
			}
		}
	}

	log.Print("Lies set")

	blacklist, _ := CreateBlacklist(inconsistentChain, 0, false, false, 0, withSuspects)

	log.Print("Create blacklist")

	return inconsistentChain, &blacklist, nodeLieMap, nil
}

func ContainsInt(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Get_M_subsets_of_K_liars_out_of_N_nodes does exactly what the name says with numbers as nodes
func GetMSubsetsOfKLiarsOutOfNNodes(M int, K int, N int) [][]int {
	superSet := makeRange(N)
	current := make([]int, 0)
	solution := make([][]int, 0)
	solution = getSubsets(superSet, K, 0, current, solution, M)

	return solution[:M]
}

func getSubsets(superSet []int, k int, idx int, current []int, solution [][]int, nbNeeded int) [][]int {
	if len(solution) >= nbNeeded {
		return solution
	}

	if len(current) == k {
		solution = append(solution, current)
		return solution
	}
	if idx == len(superSet) {
		return solution
	}
	x := superSet[idx]
	current = append(current, x)
	solution = getSubsets(superSet, k, idx+1, current, solution, nbNeeded)
	current = current[:len(current)-1]
	solution = getSubsets(superSet, k, idx+1, current, solution, nbNeeded)
	return solution
}

func makeRange(smallerThan int) []int {
	a := make([]int, smallerThan)
	for i := range a {
		a[i] = i
	}
	return a
}

//GetLies generates a pseudo-random set of lies to be reused during fixed lie testing
func GetLies(nbLiars int, nbVictims int, lowerBound int, upperBound int) []int {
	lies := make([]int, 0)
	for i := 0; i < nbLiars; i++ {
		for j := 0; j < (nbVictims - i); j++ {
			lat := rand.Intn(upperBound-lowerBound) + lowerBound
			lies = append(lies, lat)
		}
	}
	return lies
}

func pickClusteredLiars(chain *Chain, nbLiars int, nbClusters int) [][]int {

	clusters := make([][]int, nbClusters)
	usedNodes := make(map[int]bool)

	for h := 0; h < len(chain.Blocks); h++ {
		usedNodes[h] = false
	}

	for i := 0; i < nbClusters; i++ {
		//pick node from chain and get its closest neighbors (sort latencies) (without already used ones)
		if usedNodes[i] == false {
			latencyMap := chain.Blocks[i].Latencies
			sorting := make(map[int]int)
			lats := make([]int, 0)
			for node, lat := range latencyMap {
				intLat := int(lat.Latency)
				sorting[intLat] = nodesToNumbers(node)
				lats = append(lats, intLat)
			}

			uniqueLats := unique(lats)

			sort.Ints(uniqueLats)

			nodes := make([]int, 0)

			for _, lat := range uniqueLats[:nbLiars] {
				nodes = append(nodes, sorting[lat])
			}

			clusters[i] = nodes
			usedNodes[i] = true

		}

	}

	return clusters

}

func unique(intSlice []int) []int {
	keys := make(map[int]bool)
	list := []int{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
