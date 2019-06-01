package latencyprotocol

import (
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"

	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"time"

	sigAlg "golang.org/x/crypto/ed25519"
)

type sourceType int

const (
	random sourceType = iota
	accurate
	inaccurate
	variant
)

var distanceSuite = pairing.NewSuiteBn256()

/*Test ApproximateDistance initially by assuming
all nodes are honest, each node adds in the blockchain x distances from itself to other x nodes,
where these x nodes are randomly chosen. You can assume for now that there’s a publicly known source
of randomness that nodes use. Check the results by varying the number x and the total number of nodes N.*/

func initChain(N int, x int, src sourceType, nbLiars int, nbVictims int) (*Chain, []*NodeID) {
	local := onet.NewTCPTest(distanceSuite)
	local.Check = onet.CheckNone
	_, el, _ := local.GenTree(N, false)
	defer local.CloseAll()

	nodeIDs := make([]*NodeID, N)
	privateKeys := make([]sigAlg.PrivateKey, N)

	for h := 0; h < N; h++ {
		pub, priv, err := sigAlg.GenerateKey(nil)
		if err != nil {
			return nil, nil
		}
		nodeIDs[h] = &NodeID{el.List[h], pub}
		privateKeys[h] = priv
	}

	chain := Chain{[]*Block{}, []byte("testBucket")}

	for i := 0; i < N; i++ {
		latencies := make(map[string]ConfirmedLatency)
		nbLatencies := 0
		for j := 0; j < N && nbLatencies < x; j++ {
			if i != j {
				nbLatencies++
				switch src {
				case random:
					latencies[string(nodeIDs[j].PublicKey)] = ConfirmedLatency{
						time.Duration((rand.Intn(300-20) + 20)),
						nil,
						time.Now(),
						nil,
					}
				case accurate:
					latencies[string(nodeIDs[j].PublicKey)] =
						ConfirmedLatency{
							time.Duration(10 * (i + j + 1)),
							nil,
							time.Now(),
							nil,
						}
				case variant:
					//adapt to percentage of distance
					latencies[string(nodeIDs[j].PublicKey)] = ConfirmedLatency{
						time.Duration(10 * (i + j + 1 + rand.Intn(5))),
						nil,
						time.Now(),
						nil,
					}
				case inaccurate:
					if i < nbLiars && (N-nbVictims) <= j {
						latencies[string(nodeIDs[j].PublicKey)] = ConfirmedLatency{
							time.Duration(((i * 10000) + j + 1)),
							nil,
							time.Now(),
							nil,
						}
					} else {
						if j < nbLiars && (N-nbVictims) <= i {
							latencies[string(nodeIDs[j].PublicKey)] = ConfirmedLatency{
								time.Duration(((j * 10000) + i + 1)),
								nil,
								time.Now(),
								nil,
							}
						} else {
							latencies[string(nodeIDs[j].PublicKey)] =
								ConfirmedLatency{
									time.Duration(10 * (i + j + 1)),
									nil,
									time.Now(),
									nil,
								}
						}
					}

				}

			}
		}
		chain.Blocks = append(chain.Blocks, &Block{nodeIDs[i], latencies})
	}

	return &chain, nodeIDs

}

func numbersToNodes(n int) string {
	return "N" + strconv.Itoa(n)
}

func numberArrayToNodeArray(numbers []int) []string {
	nbNodes := len(numbers)
	nodes := make([]string, nbNodes)
	for i := 0; i < nbNodes; i++ {
		nodes[i] = numbersToNodes(numbers[i])

	}
	return nodes
}

func nodesToNumbers(node string) int {
	nb, _ := strconv.Atoi(string(node[1:]))
	return nb
}

func simpleChain(nbNodes int) (*Chain, []sigAlg.PublicKey) {
	blocks := make([]*Block, nbNodes)
	nodes := make([]sigAlg.PublicKey, nbNodes)

	for i := 0; i < nbNodes; i++ {
		latencies := make(map[string]ConfirmedLatency)

		for j := 0; j < nbNodes; j++ {
			if j != i {
				latencies[numbersToNodes(j)] = ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil}
			}
		}

		block := &Block{
			ID: &NodeID{
				ServerID:  nil,
				PublicKey: sigAlg.PublicKey(numbersToNodes(i)),
			},
			Latencies: latencies,
		}

		blocks[i] = block

		nodes[i] = sigAlg.PublicKey([]byte(numbersToNodes(i)))

	}

	chain := &Chain{
		Blocks:     blocks,
		BucketName: []byte("TestBucket"),
	}

	return chain, nodes
}

//func consistentChain(nbNodes int) (*Chain, []sigAlg.PublicKey) {
func consistentChain(nbNodes int, startIndex int) (*Chain, []string) {

	blocks := make([]*Block, nbNodes)
	nodes := make([]string, nbNodes)

	for i := 0; i < nbNodes; i++ {
		latencies := make(map[string]ConfirmedLatency)

		for j := 0; j < nbNodes; j++ {
			if j != i {
				lat := rand.Intn(500) + 500
				latencies[numbersToNodes(j+startIndex)] = ConfirmedLatency{time.Duration(lat), nil, time.Now(), nil}
			}
		}

		block := &Block{
			ID: &NodeID{
				ServerID:  nil,
				PublicKey: sigAlg.PublicKey(numbersToNodes(i + startIndex)),
			},
			Latencies: latencies,
		}

		blocks[i] = block

		//nodes[i] = sigAlg.PublicKey([]byte(numbersToNodes(i)))
		nodes[i] = numbersToNodes(i + startIndex)

	}

	chain := &Chain{
		Blocks:     blocks,
		BucketName: []byte("TestBucket"),
	}

	return chain, nodes
}

func setLiarAndVictim(chain *Chain, liar string, victim string, latency time.Duration) {
	chain.Blocks[nodesToNumbers(liar)].Latencies[victim] = ConfirmedLatency{time.Duration(latency * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[nodesToNumbers(victim)].Latencies[liar] = ConfirmedLatency{time.Duration(latency * time.Nanosecond), nil, time.Now(), nil}

}

func deleteLatency(chain *Chain, node1 string, node2 string) {
	delete(chain.Blocks[nodesToNumbers(node1)].Latencies, node2)
	delete(chain.Blocks[nodesToNumbers(node2)].Latencies, node1)
}

func checkBlacklistWithRemovedLatencies(chain *Chain, nodes []sigAlg.PublicKey, withSuspects bool) string {

	str := ""
	recap := "\nRecap: \n"

	delta := time.Duration(0)
	threshold := UpperThreshold(len(chain.Blocks))

	originalBlacklist, _ := CreateBlacklist(chain, delta, false, false, -1, withSuspects)

	if originalBlacklist.IsEmpty() {
		return "Even without removal, blacklist is empty"
	}

	originalBlack := originalBlacklist.GetBlacklistWithThreshold(threshold)

	recap += "Threshold: " + strconv.Itoa(threshold) + "\n"

	checkedNodes := make(map[string]bool, 0)

	for _, block := range chain.Blocks {
		node1Key := string(block.ID.PublicKey)
		checkedNodes[node1Key] = true

		keys2 := make([]string, 0, len(block.Latencies))
		for key := range block.Latencies {
			keys2 = append(keys2, key)
		}
		sort.Strings(keys2)

		for _, node2Key := range keys2 {
			_, nodeChecked := checkedNodes[node2Key]
			if !nodeChecked {
				deleteLatency(chain, node1Key, node2Key)
				newBlack, _ := CreateBlacklist(chain, delta, false, false, -1, withSuspects)

				str += "\nRemoving: " + node1Key + " <-> " + node2Key + originalBlacklist.PrintDifferencesTo(&newBlack)
				setLiarAndVictim(chain, node1Key, node2Key, block.Latencies[node2Key].Latency)

				recap += "Removed: " + node1Key + " <-> " + node2Key + ": "
				if !newBlack.NodesEqual(&originalBlack) {
					if newBlack.IsEmpty() {
						recap += "	* Blacklist emptied"
					} else {
						recap += "	* New Blacklist: " + originalBlack.NodesToString() + " -> " + newBlack.NodesToString()
					}
				} else {
					recap += "	* No changes"
				}

				recap += "\n"
			}

		}

	}

	return recap + str

}

func blacklistsEquivalent(a, b []sigAlg.PublicKey) bool {

	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if !containsKey(b, a[i]) {
			return false
		}
	}

	return true
}

func containsKey(s []sigAlg.PublicKey, e sigAlg.PublicKey) bool {
	for _, a := range s {
		if reflect.DeepEqual(a, e) {
			return true
		}
	}
	return false
}

func makeRange(smallerThan int) []int {
	a := make([]int, smallerThan)
	for i := range a {
		a[i] = i
	}
	return a
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

/**
public static List<Set<Integer>> getSubsets(List<Integer> superSet, int k) {
    List<Set<Integer>> res = new ArrayList<>();
    getSubsets(superSet, k, 0, new HashSet<Integer>(), res);
    return res;
}
**/
