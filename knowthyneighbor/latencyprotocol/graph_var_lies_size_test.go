/*
 This file allows us to measure how many liars are blacklisted for different magnitudes of lies.

 There are multiple configurable variables (see below)

 Once configured, the test should be run from the terminal within the latencyprotocol folder using the command:

	go test -run TestVarSizeLiesGraphCreation -timeout=24h


 The generated data can be found under python_graphs/var_lies, as can the jupyter notebooks to create the graphs
*/

package latencyprotocol

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"go.dedis.ch/onet/v3/log"
)

func TestVarSizeLiesGraphCreation(t *testing.T) {

	//configs =====================================================================================================
	withSuspects := true  //use enhanced blacklisting algorithm
	singleVictim := false //liars target single victim
	coordinated := false  //liars coordinate (their lies do not contradict each other)
	nbNodes := 100
	nbLiars := 33
	maxLatency := []int{50, 100, 600, 650, 700, 750, 800, 850, 900} //the maximum amount by which a lie deviates from the corresponding latency's true value

	verbose := true //print information about test to terminal

	//=============================================================================================================

	if int(nbNodes/3) < nbLiars {
		log.Print("Error: cannot have more than N/3 liars")
		return
	}

	nbVictims := "all"
	if singleVictim {
		nbVictims = "one"
	}

	coord := "coordinated"
	if !coordinated {
		coord = "uncoordinated"
	}

	filename := "test_" +
		strconv.Itoa(nbNodes) + "_nodes_" +
		strconv.Itoa(nbLiars) + "_liars" +
		"_attack_" + nbVictims +
		"_distance_" + strconv.Itoa(maxLatency[0]) + "_to_" + strconv.Itoa(maxLatency[len(maxLatency)-1]) +
		"_" + coord

	if withSuspects {
		filename += "_with_suspects"
	}

	err := CreateGrowingLieSizeGraphData(100, 33, filename, withSuspects, singleVictim, coordinated, maxLatency, verbose)
	if err != nil {
		log.Print(err)
	}

}

func CreateGrowingLieSizeGraphData(N int, nbLiars int, filename string,
	withSuspects bool, singleVictim bool, coordinated bool, maxLatencies []int, verbose bool) error {

	//4) Create a graph where each original latency is on the x-axis,
	//each corresponding latency actually recorded in the chain is on the y-axis,
	//and if the nodes at the ends of the latency (x,y) are in the blacklist, give it a different color.
	// 0, 1 or 2 nodes recorded as blacklisted
	//=> configure X, Y, Blacklist values for graphing, write to file

	file, err := os.Create("../../python_graphs/var_lies/data/var_lie_size/" + filename + ".csv")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	for _, maxLatency := range maxLatencies {
		_, _, blacklist, err := CreateFixedLiarHonestAndLyingNetworks(
			N, nbLiars, withSuspects, singleVictim, coordinated, maxLatency, verbose)

		if err != nil {
			return err
		}

		fmt.Fprintln(file, "max_lie_latency,is_liar,is_blacklisted")

		maxLat := strconv.Itoa(maxLatency)

		for i := 0; i < N; i++ {
			nodei := numbersToNodes(i)
			isLiar := strconv.FormatBool(i < nbLiars)
			isBlacklisted := strconv.FormatBool(blacklist.ContainsAsString(nodei))
			fmt.Fprintln(file, maxLat+","+isLiar+","+isBlacklisted)

		}
	}

	//5) Repeat 1-4 for a new chain with a different number of nodes (edited)
	return nil

}
