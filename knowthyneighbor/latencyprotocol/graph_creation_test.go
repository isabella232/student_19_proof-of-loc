package latencyprotocol

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"go.dedis.ch/onet/v3/log"
)

func TestGraphCreation(t *testing.T) {

	log.Print(time.Now())

	err := CreateGraphData(70, 23, "test_70_nodes")
	if err != nil {
		log.Print(err)
	}

	log.Print(time.Now())

}

func CreateGraphData(N int, nbLiars int, filename string) error {

	//1) Create chain with No TIVs or liars
	consistentChain, _ := consistentChain(N)
	log.Print("Created Consistent Graph")

	testBlacklist, _ := CreateBlacklist(consistentChain, 0, false)

	log.Print("Created Blacklist for consistent")

	if !testBlacklist.IsEmpty() {
		log.Print(testBlacklist.ToString())
		return errors.New("Original graph has triangle inequality violations")
	}

	//2) Modify some of the latencies so they might no longer be consistent
	inconsistentChain := consistentChain.Copy()
	log.Print("Copied Consistent Graph")

	for n1 := 0; n1 < nbLiars; n1++ {

		log.Print("Liar: " + numbersToNodes(n1))

		for n2 := nbLiars; n2 < N; n2++ {
			if n1 != n2 {

				oldLatency := int(consistentChain.Blocks[n1].Latencies[numbersToNodes(n2)].Latency.Nanoseconds())

				var newLatency int
				adder := 7000 //rand.Intn(7000) + 30

				newLatency = oldLatency + adder
				/*sign := rand.Intn(1)

				if sign == 0 && oldLatency > adder {
					newLatency = (oldLatency - adder)
				} else {
					newLatency = (oldLatency + adder)
				}*/

				setLiarAndVictim(inconsistentChain, numbersToNodes(n1), numbersToNodes(n2), time.Duration(newLatency))
			}
		}
	}

	log.Print("Set lies")

	//3) Create the blacklist of the chain
	blacklist, _ := CreateBlacklist(inconsistentChain, 0, true)

	print(blacklist.ToStringFake())

	log.Print("Create blacklist")

	//4) Create a graph where each original latency is on the x-axis,
	//each corresponding latency actually recorded in the chain is on the y-axis,
	//and if the nodes at the ends of the latency (x,y) are in the blacklist, give it a different color.
	// 0, 1 or 2 nodes recorded as blacklisted
	//=> configure X, Y, Blacklist values for graphing, write to file

	file, err := os.Create("graphs/" + filename + ".csv")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	fmt.Fprintln(file, "true_latency,recorded_latency,blacklist_status,")

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
