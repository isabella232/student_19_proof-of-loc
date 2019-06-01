/*
 This file allows us to measure how many liars are blacklisted, depending on the type of lies told.

 There are multiple configurable variables (see below)

 Once configured, the test should be run from the terminal within the latencyprotocol folder using the command:

	go test -run TestVarLiesGraphCreation -timeout=24h


 The generated data can be found under python_graphs/var_lies, as can the jupyter notebooks to create the graphs
*/

package latencyprotocol

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"go.dedis.ch/onet/v3/log"
)

func TestVarLiesGraphCreation(t *testing.T) {

	//configs =====================================================================================================
	linear := false       //collect data as sum or as percentage
	withSuspects := true  //use enhanced blacklisting algorithm
	singleVictim := false //liars target single victim
	coordinated := false  //liars coordinate (their lies do not contradict each other)
	nbNodes := 100
	nbLiars := 33
	maxLatency := 20000 //the maximum amount by which a lie deviates from the corresponding latency's true value

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
		"_distance_" + strconv.Itoa(maxLatency) +
		"_" + coord

	if withSuspects {
		filename += "_with_suspects"
	}

	var err error
	if linear {
		err = CreateFixedLiarLinearGraphData(nbNodes, nbLiars, filename, withSuspects, singleVictim, coordinated, maxLatency, verbose)
	} else {
		err = CreateFixedLiarPercentageGraphData(100, 33, filename, withSuspects, singleVictim, coordinated, maxLatency, verbose)
	}

	if err != nil {
		log.Print(err)
	}

}

func CreateFixedLiarLinearGraphData(N int, nbLiars int, filename string,
	withSuspects bool, singleVictim bool, coordinated bool, maxLatency int, verbose bool) error {

	consistentChain, inconsistentChain, blacklist, err := createFixedLiarHonestAndLyingNetworks(
		N, nbLiars, withSuspects, singleVictim, coordinated, maxLatency, verbose)

	if err != nil {
		return err
	}

	//4) Create a graph where each original latency is on the x-axis,
	//each corresponding latency actually recorded in the chain is on the y-axis,
	//and if the nodes at the ends of the latency (x,y) are in the blacklist, give it a different color.
	// 0, 1 or 2 nodes recorded as blacklisted
	//=> configure X, Y, Blacklist values for graphing, write to file

	file, err := os.Create("../../python_graphs/var_lies/data/linear/" + filename + ".csv")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	fmt.Fprintln(file, "true_latency,recorded_latency,blacklist_status")

	for i := 0; i < N; i++ {
		blacklistStatusBlock := 0
		if blacklist.ContainsAsString(numbersToNodes(i)) {
			blacklistStatusBlock++
		}

		for j := i + 1; j < N; j++ {
			blacklistStatus := 0
			nodej := numbersToNodes(j)
			if blacklist.ContainsAsString(nodej) {
				blacklistStatus++
			}
			real := strconv.Itoa(int(consistentChain.Blocks[i].Latencies[nodej].Latency))
			recorded := strconv.Itoa(int(inconsistentChain.Blocks[i].Latencies[nodej].Latency))
			status := strconv.Itoa(blacklistStatus + blacklistStatusBlock)
			fmt.Fprintln(file, real+","+recorded+","+status)

		}
	}

	//5) Repeat 1-4 for a new chain with a different number of nodes (edited)
	return nil

}

func CreateFixedLiarPercentageGraphData(N int, nbLiars int, filename string,
	withSuspects bool, singleVictim bool, coordinated bool, maxLatency int, verbose bool) error {

	//4) Create a graph where each original latency is on the x-axis,
	//each corresponding latency actually recorded in the chain is on the y-axis,
	//and if the nodes at the ends of the latency (x,y) are in the blacklist, give it a different color.
	// 0, 1 or 2 nodes recorded as blacklisted
	//=> configure X, Y, Blacklist values for graphing, write to file

	consistentChain, inconsistentChain, blacklist, err := createFixedLiarHonestAndLyingNetworks(
		N, nbLiars, withSuspects, singleVictim, coordinated, maxLatency, verbose)
	thresh := UpperThreshold(N)
	threshold := strconv.Itoa(thresh)

	if err != nil {
		return err
	}

	file, err := os.Create("../../python_graphs/var_lies/data/percentage/" + filename + ".csv")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	fmt.Fprintln(file, "node_1,node_2,lie_percentage,nb_strikes_1,nb_strikes_2,threshold,blacklisted_1,blacklisted_2")

	for i := 0; i < N; i++ {
		nodei := numbersToNodes(i)

		for j := 0; j < N; j++ {
			if i != j {
				nodej := numbersToNodes(j)
				real := int(consistentChain.Blocks[i].Latencies[nodej].Latency)
				recorded := int(inconsistentChain.Blocks[i].Latencies[nodej].Latency)
				percentage := strconv.FormatFloat(math.Abs(float64(recorded-real))/float64(real), 'f', 2, 64)
				nbStrikes1 := strconv.Itoa(blacklist.NbStrikesOf(nodei))
				nbStrikes2 := strconv.Itoa(blacklist.NbStrikesOf(nodej))
				blacklisted1 := strconv.FormatBool(blacklist.ContainsAsString(nodei))
				blacklisted2 := strconv.FormatBool(blacklist.ContainsAsString(nodej))
				fmt.Fprintln(file, nodei+","+nodej+","+percentage+","+nbStrikes1+","+nbStrikes2+","+threshold+","+blacklisted1+","+blacklisted2)
			}

		}
	}

	//5) Repeat 1-4 for a new chain with a different number of nodes (edited)
	return nil

}

func createFixedLiarHonestAndLyingNetworks(N int, nbLiars int, withSuspects bool, singleVictim bool, coordinated bool, maxLatency int, verbose bool) (
	*Chain, *Chain, *Blacklistset, error) {

	//1) Create chain with No TIVs or liars
	consistentChain, _ := consistentChain(N, 0)
	log.Print("Created Consistent Graph")

	testBlacklist, _ := CreateBlacklist(consistentChain, 0, false, true, 0, withSuspects)

	log.Print("Created Blacklist for consistent")

	if !testBlacklist.IsEmpty() {
		log.Print(testBlacklist.ToString())
		return nil, nil, nil, errors.New("Original graph has triangle inequality violations")
	}

	//2) Modify some of the latencies so they might no longer be consistent
	inconsistentChain := consistentChain.Copy()
	log.Print("Copied Consistent Graph")

	//All liars target 1 victim

	if singleVictim {
		victim := nbLiars
		for n1 := 0; n1 < nbLiars; n1++ {
			oldLatency := int(consistentChain.Blocks[n1].Latencies[numbersToNodes(victim)].Latency.Nanoseconds())

			var newLatency int
			if coordinated {
				newLatency = oldLatency + maxLatency
			} else {
				adder := rand.Intn(maxLatency)
				sign := rand.Intn(2)

				if sign == 0 && oldLatency > adder {
					newLatency = (oldLatency - adder)
				} else {
					newLatency = (oldLatency + adder)
				}
			}

			setLiarAndVictim(inconsistentChain, numbersToNodes(n1), numbersToNodes(victim), time.Duration(newLatency))

		}
	} else {

		for n1 := 0; n1 < nbLiars; n1++ {

			log.Print("Liar: " + numbersToNodes(n1))

			//liars not attacking each other: n2 := nbLiars
			for n2 := 0; n2 < N; n2++ {
				if n1 != n2 {

					oldLatency := int(consistentChain.Blocks[n1].Latencies[numbersToNodes(n2)].Latency.Nanoseconds())

					var newLatency int
					if coordinated {
						newLatency = oldLatency + maxLatency
					} else {
						adder := rand.Intn(maxLatency)
						sign := rand.Intn(2)

						if sign == 0 && oldLatency > adder {
							newLatency = (oldLatency - adder)
						} else {
							newLatency = (oldLatency + adder)
						}
					}

					setLiarAndVictim(inconsistentChain, numbersToNodes(n1), numbersToNodes(n2), time.Duration(newLatency))
				}
			}
		}
	}

	log.Print("Lies set")

	//3) Create the blacklist of the chain
	blacklist, _ := CreateBlacklist(inconsistentChain, 0, verbose, false, 0, withSuspects)

	log.Print("Create blacklist")

	return consistentChain, inconsistentChain, &blacklist, nil

}
