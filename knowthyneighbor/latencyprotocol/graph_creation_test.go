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

func TestFixedLiarGraphCreation(t *testing.T) {

	err := CreateFixedLiarPercentageGraphData(100, 33, "test_100_nodes_attack_all_90000")
	if err != nil {
		log.Print(err)
	}

}

func CreateFixedLiarLinearGraphData(N int, nbLiars int, filename string) error {

	consistentChain, inconsistentChain, blacklist, err := createFixedLiarHonestAndLyingNetworks(N, nbLiars)

	if err != nil {
		return err
	}

	//4) Create a graph where each original latency is on the x-axis,
	//each corresponding latency actually recorded in the chain is on the y-axis,
	//and if the nodes at the ends of the latency (x,y) are in the blacklist, give it a different color.
	// 0, 1 or 2 nodes recorded as blacklisted
	//=> configure X, Y, Blacklist values for graphing, write to file

	file, err := os.Create("../../python_graphs/fixed_liars/data/linear/" + filename + ".csv")
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

func CreateFixedLiarPercentageGraphData(N int, nbLiars int, filename string) error {

	//4) Create a graph where each original latency is on the x-axis,
	//each corresponding latency actually recorded in the chain is on the y-axis,
	//and if the nodes at the ends of the latency (x,y) are in the blacklist, give it a different color.
	// 0, 1 or 2 nodes recorded as blacklisted
	//=> configure X, Y, Blacklist values for graphing, write to file

	consistentChain, inconsistentChain, blacklist, err := createFixedLiarHonestAndLyingNetworks(N, nbLiars)
	thresh := UpperThreshold(N)
	threshold := strconv.Itoa(thresh)

	if err != nil {
		return err
	}

	file, err := os.Create("../../python_graphs/data/percentage/" + filename + ".csv")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	fmt.Fprintln(file, "node_1,node_2,lie_percentage,nb_strikes_1,nb_strikes_2,threshold")

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
				fmt.Fprintln(file, nodei+","+nodej+","+percentage+","+nbStrikes1+","+nbStrikes2+","+threshold)
			}

		}
	}

	//5) Repeat 1-4 for a new chain with a different number of nodes (edited)
	return nil

}

func createFixedLiarHonestAndLyingNetworks(N int, nbLiars int) (*Chain, *Chain, *Blacklistset, error) {

	//1) Create chain with No TIVs or liars
	consistentChain, _ := consistentChain(N)
	log.Print("Created Consistent Graph")

	testBlacklist, _ := CreateBlacklist(consistentChain, 0, false, true, 0)

	log.Print("Created Blacklist for consistent")

	if !testBlacklist.IsEmpty() {
		log.Print(testBlacklist.ToString())
		return nil, nil, nil, errors.New("Original graph has triangle inequality violations")
	}

	//2) Modify some of the latencies so they might no longer be consistent
	inconsistentChain := consistentChain.Copy()
	log.Print("Copied Consistent Graph")

	//All liars target 1 victim
	/*victim := nbLiars
	for n1 := 0; n1 < nbLiars; n1++ {
		oldLatency := int(consistentChain.Blocks[n1].Latencies[numbersToNodes(victim)].Latency.Nanoseconds())

		var newLatency int
		//coordinated attack:
		newLatency = oldLatency + 7000
		/*adder := rand.Intn(7000)
		sign := rand.Intn(2)

		if sign == 0 && oldLatency > adder {
			newLatency = (oldLatency - adder)
		} else {
			newLatency = (oldLatency + adder)
		}

		setLiarAndVictim(inconsistentChain, numbersToNodes(n1), numbersToNodes(victim), time.Duration(newLatency))

	}*/

	for n1 := 0; n1 < nbLiars; n1++ {

		log.Print("Liar: " + numbersToNodes(n1))

		//liars not attacking each other: n2 := nbLiars
		for n2 := 0; n2 < N; n2++ {
			if n1 != n2 {

				oldLatency := int(consistentChain.Blocks[n1].Latencies[numbersToNodes(n2)].Latency.Nanoseconds())

				var newLatency int
				//coordinated attack: newLatency = oldLatency + 7000
				//adder := rand.Intn(7000)
				//outrageous lies:
				adder := rand.Intn(90000)
				//very outrageous lies: adder := rand.Intn(100000)

				sign := rand.Intn(2)

				if sign == 0 && oldLatency > adder {
					newLatency = (oldLatency - adder)
				} else {
					newLatency = (oldLatency + adder)
				}

				setLiarAndVictim(inconsistentChain, numbersToNodes(n1), numbersToNodes(n2), time.Duration(newLatency))
			}
		}
	}

	log.Print("Lies set")

	//3) Create the blacklist of the chain
	blacklist, _ := CreateBlacklist(inconsistentChain, 0, true, true, 0)

	log.Print("Create blacklist")

	return consistentChain, inconsistentChain, &blacklist, nil

}
