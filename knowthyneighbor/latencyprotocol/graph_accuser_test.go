package latencyprotocol

import (
	"math/rand"
	"testing"
	"time"

	"go.dedis.ch/onet/v3/log"
	sigAlg "golang.org/x/crypto/ed25519"
)

/**
Experiments on a node trying to make itself closer to multiple clusters of nodes. Take 2 clusters.
The size of the clusters, the distance between the clusters.
Keep the cluster sizes the same, vary the distance between them
Keep the distance the same, keep the sum of nodes in clusters the same, vary the sizes of these clusters
Output: is the node detected or not

Case 1:
**/

func Test_accuser_analysis(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())

	//create clustered network: reaches max possible strikes at 1978 distance
	//distance := binarySearch(200000, N, clusterSizes)

	distance := 100000

	N := 9
	//nbClusters := 2
	clusterSizes := []int{3, 3, 3}

	consistentChain, _, _ := create_graph_with_C_clusters(len(clusterSizes), clusterSizes, distance)
	//test
	inconsistentChain := set_lies_to_clusters(0, consistentChain)

	thresh := UpperThreshold(N)
	//threshold := strconv.Itoa(thresh)
	blacklist, err := CreateBlacklist(inconsistentChain, 0, false, true, thresh)
	if err != nil {
		log.Print(err)
	}
	unthresholded, err := CreateBlacklist(inconsistentChain, 0, false, true, 0)
	if err != nil {
		log.Print(err)
	}

	suspects := checkStrikes(&unthresholded, N)

	for _, suspect := range suspects {
		probablyLiar := checkSuspect(inconsistentChain, suspect, N)
		if probablyLiar {
			log.Print("Probably liar: " + suspect)
		} else {
			log.Print("Probably victim: " + suspect)
		}
	}

	log.Print("Actual blacklist: ")
	log.Print(blacklist.ToString())

}

func BlacklistEnhancement(chain *Chain, N int) []string {
	unthresholded, err := CreateBlacklist(chain, 0, false, true, 0)
	if err != nil {
		log.Print(err)
	}

	suspects := checkStrikes(&unthresholded, N)

	newBlacklistees := make([]string, 0)

	for _, suspect := range suspects {
		probablyLiar := checkSuspect(chain, suspect, N)
		if probablyLiar {
			newBlacklistees = append(newBlacklistees, suspect)
		}
	}
	return newBlacklistees
}

func checkStrikes(strikelist *Blacklistset, N int) []string {
	average := 0
	for _, nbStrikes := range strikelist.Strikes {
		average += nbStrikes
	}
	average = average / N

	suspicious := make([]string, 0)
	threshold := UpperThreshold(N)

	for node, nbStrikes := range strikelist.Strikes {
		if average < nbStrikes && nbStrikes < threshold {
			suspicious = append(suspicious, node)
		}
	}

	return suspicious
}

func checkSuspect(chain *Chain, suspect string, N int) bool {

	blockMapper := make(map[string]*Block)

	blacklist := NewBlacklistset()

	for _, block := range chain.Blocks {
		blockMapper[string(block.ID.PublicKey)] = block
	}

	//for each node B
	//for each node C
	//for each node D
	/* Check B -> C, B -> D, C -> D
	* if triangle of lengths does not result in realistic angles (rule of 3 for triangles),
	B, C or D needs to be blacklisted -> add (B,C, D) to a "suspicious" list and keep checking B
	*/

	suspectBlock := blockMapper[suspect]

	for Cstring := range suspectBlock.Latencies {
		if Cstring != suspect {
			CBlock, CHere := blockMapper[Cstring]

			if CHere {

				for Dstring := range suspectBlock.Latencies {
					if Dstring != Cstring && Dstring != suspect {
						DBlock, DHere := blockMapper[Dstring]

						if DHere {

							BtoD, BtoDHere := suspectBlock.getLatency(DBlock)
							BtoC, BtoCHere := suspectBlock.getLatency(CBlock)
							CtoD, CtoDHere := CBlock.getLatency(DBlock)

							if BtoDHere && BtoCHere && CtoDHere && !TriangleInequalitySatisfiedInt(int(BtoD), int(BtoC), int(CtoD)) {

								blacklist.Add(sigAlg.PublicKey([]byte(suspect)))
								blacklist.Add(sigAlg.PublicKey([]byte(Cstring)))
								blacklist.Add(sigAlg.PublicKey([]byte(Dstring)))
							}

						}
					}

				}

			}
		}
	}

	//non-accusers: nodes that do not give more than N/3 strikes (the N/3 might be given by the liars)
	nbNonAccusers := 0
	accuserThreshold := int(N/3) * 6
	for _, nbStrikes := range blacklist.Strikes {
		if nbStrikes <= accuserThreshold {
			nbNonAccusers++
		}
	}
	nbNonAccusersNeeded := 2 * (N / 3)
	return nbNonAccusers >= nbNonAccusersNeeded
}
