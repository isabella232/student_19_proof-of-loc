package latencyprotocol

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"go.dedis.ch/onet/v3/log"
)

func TestGraphCreation(t *testing.T) {

	/*log.Print("100")

	err := CreateGraphData(100, 33, "test_100_nodes_attack_all")
	if err != nil {
		log.Print(err)
	}*/

	log.Print("200")

	err := CreateGraphData(200, 66, "test")
	if err != nil {
		log.Print(err)
	}

	/*log.Print("500")

	err = CreateGraphData(500, 166, "test_500_nodes_attack_all")
	if err != nil {
		log.Print(err)
	}

	/*err = CreateGraphData(1000, 333, "test_1000_nodes")
	if err != nil {
		log.Print(err)
	}

	err = CreateGraphData(2000, 666, "test_2000_nodes")
	if err != nil {
		log.Print(err)
	}

	err = CreateGraphData(3000, 1000, "test_3000_nodes")
	if err != nil {
		log.Print(err)
	}

	err = CreateGraphData(5000, 1666, "test_5000_nodes")
	if err != nil {
		log.Print(err)
	}*/

}

func CreateGraphData(N int, nbLiars int, filename string) error {

	//1) Create chain with No TIVs or liars
	consistentChain, _ := consistentChain(N)
	log.Print("Created Consistent Graph")

	testBlacklist, _ := CreateBlacklist(consistentChain, 0, false, true, 0)

	log.Print("Created Blacklist for consistent")

	if !testBlacklist.IsEmpty() {
		log.Print(testBlacklist.ToString())
		return errors.New("Original graph has triangle inequality violations")
	}

	//2) Modify some of the latencies so they might no longer be consistent
	inconsistentChain := consistentChain.Copy()
	log.Print("Copied Consistent Graph")

	//All liars target 1 victim
	victim := nbLiars
	for n1 := 0; n1 < nbLiars; n1++ {
		oldLatency := int(consistentChain.Blocks[n1].Latencies[numbersToNodes(victim)].Latency.Nanoseconds())

		var newLatency int
		//coordinated attack: newLatency = oldLatency + 7000
		adder := rand.Intn(7000)
		sign := rand.Intn(2)

		if sign == 0 && oldLatency > adder {
			newLatency = (oldLatency - adder)
		} else {
			newLatency = (oldLatency + adder)
		}

		setLiarAndVictim(inconsistentChain, numbersToNodes(n1), numbersToNodes(victim), time.Duration(newLatency))

	}

	/*for n1 := 0; n1 < nbLiars; n1++ {

		log.Print("Liar: " + numbersToNodes(n1))

		//liars not attacking each other: n2 := nbLiars
		for n2 := 0; n2 < N; n2++ {
			if n1 != n2 {

				oldLatency := int(consistentChain.Blocks[n1].Latencies[numbersToNodes(n2)].Latency.Nanoseconds())

				var newLatency int
				//coordinated attack: newLatency = oldlatency + 7000
				adder := rand.Intn(7000)
				sign := rand.Intn(2)

				if sign == 0 && oldLatency > adder {
					newLatency = (oldLatency - adder)
				} else {
					newLatency = (oldLatency + adder)
				}

				setLiarAndVictim(inconsistentChain, numbersToNodes(n1), numbersToNodes(n2), time.Duration(newLatency))
			}
		}
	}*/

	log.Print("Lies set")

	//3) Create the blacklist of the chain
	blacklist, _ := CreateBlacklist(inconsistentChain, 0, true, false, -1)

	print(blacklist.ToStringFake())

	log.Print("Create blacklist")

	//4) Create a graph where each original latency is on the x-axis,
	//each corresponding latency actually recorded in the chain is on the y-axis,
	//and if the nodes at the ends of the latency (x,y) are in the blacklist, give it a different color.
	// 0, 1 or 2 nodes recorded as blacklisted
	//=> configure X, Y, Blacklist values for graphing, write to file

	file, err := os.Create("../../python_graphs/data/" + filename + ".csv")
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
