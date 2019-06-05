package latencyprotocol

import (
	"log"
	"sort"

	"strconv"
	"time"

	sigAlg "golang.org/x/crypto/ed25519"
)

/*
CreateBlacklist iterates through a chain and for each block checks if the latencies qiven by its node make sense
If they do not, the node is added to a blacklist of nodes not to be trusted

Warning: for now, the following assumptions are made:
	*all nodes give latencies to all other nodes (except themselves)
	*all latencies are symmetric (A -> B == B -> A)
*/
func CreateBlacklist(chain *Chain, delta time.Duration, verbose bool, threshGiven bool, threshold int, withSuspect bool) (Blacklistset, error) {

	N := len(chain.Blocks)

	if !threshGiven {
		threshold = UpperThreshold(N)
	}

	if verbose {
		log.Print("Threshold: " + strconv.Itoa(threshold))
	}
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

	for _, BBlock := range chain.Blocks {

		Bstring := string(BBlock.ID.PublicKey)

		for Cstring := range BBlock.Latencies {
			if Cstring != Bstring {
				CBlock, CHere := blockMapper[Cstring]

				if CHere {

					for Dstring := range BBlock.Latencies {
						if Dstring != Cstring && Dstring != Bstring {
							DBlock, DHere := blockMapper[Dstring]

							if DHere {

								BtoD, BtoDHere := BBlock.getLatency(DBlock)
								BtoC, BtoCHere := BBlock.getLatency(CBlock)
								CtoD, CtoDHere := CBlock.getLatency(DBlock)

								if BtoDHere && BtoCHere && CtoDHere && !TriangleInequalitySatisfiedInt(int(BtoD), int(BtoC), int(CtoD)) {

									blacklist.Add(sigAlg.PublicKey([]byte(Bstring)))
									blacklist.Add(sigAlg.PublicKey([]byte(Cstring)))
									blacklist.Add(sigAlg.PublicKey([]byte(Dstring)))

								}

							}
						}

					}
				}
			}
		}
	}

	if verbose {
		log.Print("Before Thresholding: ")
		log.Print(blacklist.ToString())
	}

	threshBlacklist := blacklist.GetBlacklistWithThreshold(threshold)
	if withSuspect == true {
		suspects := BlacklistEnhancement(chain, N)
		for _, suspect := range suspects {
			if !threshBlacklist.ContainsAsString(suspect) {
				threshBlacklist.AddWithStrikesStringKey(suspect, 1)
			}
		}
	}

	if verbose {
		log.Print("After Thresholding: ")
		log.Print(threshBlacklist.ToString())
	}

	return threshBlacklist, nil

}

//UpperThreshold returns the maximum number of strikes a victim node can get
func UpperThreshold(N int) int {
	third := float64(N) / 3
	return int(int(third)*int(N-1)) * 6
	//Multiply by 6 because that's how often a triangle will be tested
	//return (N / 3) * (N - (N / 3) - 1)

}

//BlacklistEnhancement enhanced the basic blacklisting triangle inequality algorithm by checking strike patterns
func BlacklistEnhancement(chain *Chain, N int) []string {
	unthresholded, err := CreateBlacklist(chain, 0, false, true, 0, false)
	if err != nil {
		log.Print(err)
	}

	suspects := checkStrikes(&unthresholded, N)

	newBlacklistees := make([]string, 0)

	for _, suspect := range suspects {
		probablyLiar := SuspectIsLiar(chain, suspect, N)
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

	sort.Strings(suspicious)
	return suspicious
}

//SuspectIsLiar checks whether a node can be blacklisted based on the strike patterns surrounding it
func SuspectIsLiar(chain *Chain, suspect string, N int) bool {

	blockMapper := make(map[string]*Block)

	blacklist := NewBlacklistset()

	for _, block := range chain.Blocks {
		blockMapper[string(block.ID.PublicKey)] = block
	}

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
	nbAccusers := 0
	accuserThreshold := int(N / 3)
	for node, nbStrikes := range blacklist.Strikes {
		if node != suspect && int(nbStrikes/2) > accuserThreshold { //divide by 2, because each triangle counted twice
			nbAccusers++
		}
	}
	nbNonAccusers := N - nbAccusers
	nbNonAccusersNeeded := int((2 * N / 3))
	//nbNonAccusersNeeded := int((N / 3)) + 1 //try this

	//if we cannot find 2N/3 nodes willing to not accuse for the suspect, the suspect is a liar
	return nbNonAccusers < nbNonAccusersNeeded
}
