package latencyprotocol

import (
	"math/rand"
	"testing"
	"time"

	"go.dedis.ch/onet/v3/log"
)

//Problem: none of the tests create chains that blacklist nodes even with all latencies given
//Solution: this is the test for coordinated, proving detection does not work in this case
//Create other file with uncoherent connections instead, where it should work.
func TestRandomAttack1(t *testing.T) {
	N := 4

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N0", "N3", 1000)
	setLiarAndVictim(chain, "N0", "N2", 40)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs))

}

func TestRandomAttack2(t *testing.T) {
	N := 8

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N0", "N1", 25)
	setLiarAndVictim(chain, "N1", "N2", 250)
	setLiarAndVictim(chain, "N0", "N3", 25)
	setLiarAndVictim(chain, "N1", "N4", 3)
	setLiarAndVictim(chain, "N0", "N5", 25)
	setLiarAndVictim(chain, "N1", "N6", 900)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs))

}

func TestRandomAttack3(t *testing.T) {
	N := 9

	chain, nodeIDs := simpleChain(N)

	//Liars: N0, N1, N2

	setLiarAndVictim(chain, "N0", "N3", 250)
	setLiarAndVictim(chain, "N0", "N4", 75)
	setLiarAndVictim(chain, "N0", "N5", 25)
	setLiarAndVictim(chain, "N1", "N3", 5)
	setLiarAndVictim(chain, "N1", "N4", 500)
	setLiarAndVictim(chain, "N1", "N5", 35)
	setLiarAndVictim(chain, "N2", "N3", 16)
	setLiarAndVictim(chain, "N2", "N4", 0)
	setLiarAndVictim(chain, "N2", "N5", 5)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs))

}

func TestRandomAttack4(t *testing.T) {
	N := 15

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N0", "N10", 25)
	setLiarAndVictim(chain, "N1", "N10", 25)
	setLiarAndVictim(chain, "N2", "N10", 25)
	setLiarAndVictim(chain, "N3", "N10", 25)

	setLiarAndVictim(chain, "N0", "N6", 25)
	setLiarAndVictim(chain, "N1", "N6", 25)
	setLiarAndVictim(chain, "N2", "N6", 25)
	setLiarAndVictim(chain, "N3", "N6", 25)

	setLiarAndVictim(chain, "N0", "N7", 25)
	setLiarAndVictim(chain, "N1", "N7", 25)
	setLiarAndVictim(chain, "N2", "N7", 25)
	setLiarAndVictim(chain, "N3", "N7", 25)

	setLiarAndVictim(chain, "N0", "N8", 25)
	setLiarAndVictim(chain, "N1", "N8", 25)
	setLiarAndVictim(chain, "N2", "N8", 25)
	setLiarAndVictim(chain, "N3", "N8", 25)

	setLiarAndVictim(chain, "N0", "N9", 25)
	setLiarAndVictim(chain, "N1", "N9", 25)
	setLiarAndVictim(chain, "N2", "N9", 25)
	setLiarAndVictim(chain, "N3", "N9", 25)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs))

}

func TestRandomAttack5(t *testing.T) {
	N := 15

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N0", "N10", 25)
	setLiarAndVictim(chain, "N1", "N10", 25)
	setLiarAndVictim(chain, "N2", "N10", 25)

	setLiarAndVictim(chain, "N0", "N6", 25)
	setLiarAndVictim(chain, "N1", "N6", 25)
	setLiarAndVictim(chain, "N2", "N6", 25)

	setLiarAndVictim(chain, "N0", "N7", 25)
	setLiarAndVictim(chain, "N1", "N7", 25)
	setLiarAndVictim(chain, "N2", "N7", 25)

	setLiarAndVictim(chain, "N0", "N8", 25)
	setLiarAndVictim(chain, "N1", "N8", 25)
	setLiarAndVictim(chain, "N2", "N8", 25)

	setLiarAndVictim(chain, "N0", "N9", 25)
	setLiarAndVictim(chain, "N1", "N9", 25)
	setLiarAndVictim(chain, "N2", "N9", 25)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs))

}

func TestRandomAttack6(t *testing.T) {
	N := 15

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N0", "N10", 25)
	setLiarAndVictim(chain, "N1", "N10", 25)
	setLiarAndVictim(chain, "N2", "N10", 25)
	setLiarAndVictim(chain, "N3", "N10", 25)

	setLiarAndVictim(chain, "N0", "N6", 25)
	setLiarAndVictim(chain, "N1", "N6", 25)
	setLiarAndVictim(chain, "N2", "N6", 25)
	setLiarAndVictim(chain, "N3", "N6", 25)

	setLiarAndVictim(chain, "N0", "N7", 25)
	setLiarAndVictim(chain, "N1", "N7", 25)
	setLiarAndVictim(chain, "N2", "N7", 25)
	setLiarAndVictim(chain, "N3", "N7", 25)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs))

}

func TestRandomAttack7(t *testing.T) {
	N := 15

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N0", "N10", 25)
	setLiarAndVictim(chain, "N1", "N10", 25)
	setLiarAndVictim(chain, "N2", "N10", 25)
	setLiarAndVictim(chain, "N3", "N10", 25)
	setLiarAndVictim(chain, "N4", "N10", 25)

	setLiarAndVictim(chain, "N0", "N6", 25)
	setLiarAndVictim(chain, "N1", "N6", 25)
	setLiarAndVictim(chain, "N2", "N6", 25)
	setLiarAndVictim(chain, "N3", "N6", 25)
	setLiarAndVictim(chain, "N4", "N6", 25)

	setLiarAndVictim(chain, "N0", "N7", 25)
	setLiarAndVictim(chain, "N1", "N7", 25)
	setLiarAndVictim(chain, "N2", "N7", 25)
	setLiarAndVictim(chain, "N3", "N7", 25)
	setLiarAndVictim(chain, "N4", "N7", 25)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs))

}
func TestRandomAttack8(t *testing.T) {
	N := 15

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N0", "N10", 25)
	setLiarAndVictim(chain, "N1", "N10", 25)
	setLiarAndVictim(chain, "N2", "N10", 25)
	setLiarAndVictim(chain, "N3", "N10", 25)
	setLiarAndVictim(chain, "N4", "N10", 25)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs))

}

func TestRandomAttack9(t *testing.T) {
	N := 15

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "N0", "N10", 25)
	setLiarAndVictim(chain, "N1", "N10", 25)
	setLiarAndVictim(chain, "N2", "N10", 25)
	setLiarAndVictim(chain, "N3", "N10", 25)

	setLiarAndVictim(chain, "N0", "N6", 25)
	setLiarAndVictim(chain, "N1", "N6", 25)
	setLiarAndVictim(chain, "N2", "N6", 25)
	setLiarAndVictim(chain, "N3", "N6", 25)

	setLiarAndVictim(chain, "N0", "N7", 25)
	setLiarAndVictim(chain, "N1", "N7", 25)
	setLiarAndVictim(chain, "N2", "N7", 25)
	setLiarAndVictim(chain, "N3", "N7", 25)

	setLiarAndVictim(chain, "N0", "N8", 25)
	setLiarAndVictim(chain, "N1", "N8", 25)
	setLiarAndVictim(chain, "N2", "N8", 25)
	setLiarAndVictim(chain, "N3", "N8", 25)

	setLiarAndVictim(chain, "N0", "N9", 25)
	setLiarAndVictim(chain, "N1", "N9", 25)
	setLiarAndVictim(chain, "N2", "N9", 25)
	setLiarAndVictim(chain, "N3", "N9", 25)

	setLiarAndVictim(chain, "N0", "N13", 25)
	setLiarAndVictim(chain, "N1", "N13", 25)
	setLiarAndVictim(chain, "N2", "N13", 25)
	setLiarAndVictim(chain, "N3", "N13", 25)

	setLiarAndVictim(chain, "N0", "N11", 25)
	setLiarAndVictim(chain, "N1", "N11", 25)
	setLiarAndVictim(chain, "N2", "N11", 25)
	setLiarAndVictim(chain, "N3", "N11", 25)

	setLiarAndVictim(chain, "N0", "N12", 25)
	setLiarAndVictim(chain, "N1", "N12", 25)
	setLiarAndVictim(chain, "N2", "N12", 25)
	setLiarAndVictim(chain, "N3", "N12", 25)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs))

}

func CreateGraphData(N int, nbLiars int) {

	//1) Create chain with No TIVs or liars
	consistentChain, _ := consistentChain(N)

	//2) Modify some of the latencies so they might no longer be consistent
	inconsistentChain := consistentChain.Copy()

	nbLiars = rand.Intn(N / 3)

	for j := 0; j < nbLiars; j++ {
		n1 := rand.Intn(N)
		n2 := rand.Intn(N)
		if n2 == n1 {
			if n1 > 0 {
				n2 = (n1 - 1)
			} else {
				n2 = (n1 + 1)
			}
		}

		oldLatency := int(consistentChain.Blocks[n1].Latencies[numbersToNodes(n2)].Latency.Nanoseconds())
		newLatency := (oldLatency * rand.Intn(N)) % N

		setLiarAndVictim(inconsistentChain, numbersToNodes(n1), numbersToNodes(n2), time.Duration(newLatency))
	}

	//3) Create the blacklist of the chain
	blacklist, _ := CreateBlacklist(inconsistentChain, 0)
	log.Print(blacklist)
	//4) Create a graph where each original latency is on the x-axis,
	//each corresponding latency actually recorded in the chain is on the y-axis,
	//and if a node at (x,y) is in the blacklist, give it a different color.
	//=> configure X, Y, Blacklist values for graphing, write to file

	//5) Repeat 1-4 for a new chain with a different number of nodes (edited)

}
