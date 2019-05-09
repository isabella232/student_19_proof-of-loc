package latencyprotocol

import (
	"math"

	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"

	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
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

func consistentChain(nbNodes int) (*Chain, []sigAlg.PublicKey) {

	latencyMap := make(map[string]int)

	for i := 0; i < nbNodes; i++ {

		for j := i + 1; j < nbNodes; j++ {
			for k := j + 1; k < nbNodes; k++ {
				n1 := numbersToNodes(i)
				n2 := numbersToNodes(j)
				n3 := numbersToNodes(k)
				pair1 := n1 + "-" + n2
				pair2 := n2 + "-" + n3
				pair3 := n1 + "-" + n3

				lat1, have1 := latencyMap[pair1]
				lat2, have2 := latencyMap[pair2]
				lat3, have3 := latencyMap[pair3]

				// 2/3 known
				if !have1 && have2 && have3 {
					floor := int(math.Abs(float64(lat3 - lat2)))
					var lat1 int
					if lat3+lat2-floor <= 0 {
						lat1 = 0
					} else {
						lat1 = rand.Intn(lat3+lat2-floor) + floor
					}
					latencyMap[pair1] = lat1
					have1 = true
				}

				if have1 && !have2 && have3 {
					floor := int(math.Abs(float64(lat1 - lat3)))
					var lat2 int
					if lat1+lat3-floor <= 0 {
						lat2 = 0
					} else {
						lat2 = rand.Intn(lat1+lat3-floor) + floor
					}
					latencyMap[pair2] = lat2
					have2 = true
				}

				if have1 && have2 && !have3 {
					floor := int(math.Abs(float64(lat1 - lat2)))
					var lat3 int
					if lat1+lat2-floor <= 0 {
						lat3 = 0
					} else {
						lat3 = rand.Intn(lat1+lat2-floor) + floor
					}
					latencyMap[pair3] = lat3
					have3 = true
				}

				// 1/3 known
				if have1 && !have2 && !have3 {
					lat2 = rand.Intn(1000)
					floor := int(math.Abs(float64(lat1 - lat2)))
					var lat3 int
					if lat1+lat2-floor <= 0 {
						lat3 = 0
					} else {
						lat3 = rand.Intn(lat1+lat2-floor) + floor
					}
					latencyMap[pair2] = lat2
					latencyMap[pair3] = lat3
					have2 = true
					have3 = true
				}

				if !have1 && have2 && !have3 {
					lat1 = rand.Intn(1000)
					var lat3 int
					floor := int(math.Abs(float64(lat1 - lat2)))
					if lat1+lat2-floor <= 0 {
						lat3 = 0
					} else {
						lat3 = rand.Intn(lat1+lat2-floor) + floor
					}
					latencyMap[pair1] = lat1
					latencyMap[pair3] = lat3
					have1 = true
					have3 = true
				}

				if !have1 && !have2 && have3 {
					lat1 = rand.Intn(1000)
					var lat2 int
					floor := int(math.Abs(float64(lat3 - lat1)))
					if lat3+lat1-floor <= 0 {
						lat2 = 0
					} else {
						lat2 = rand.Intn(lat3+lat1-floor) + floor
					}
					latencyMap[pair1] = lat1
					latencyMap[pair2] = lat2
					have1 = true
					have2 = true
				}

				// 0/3 known
				if !have1 && !have2 && !have3 {
					lat1 = rand.Intn(1000)
					lat2 = rand.Intn(1000)
					floor := int(math.Abs(float64(lat1 - lat2)))
					var lat3 int
					if lat1+lat2-floor <= 0 {
						lat3 = 0
					} else {
						lat3 = rand.Intn(lat1+lat2-floor) + floor
					}
					latencyMap[pair1] = lat1
					latencyMap[pair2] = lat2
					latencyMap[pair3] = lat3
					have1 = true
					have2 = true
					have3 = true
				}

			}
		}

	}

	blocks := make([]*Block, nbNodes)
	nodes := make([]sigAlg.PublicKey, nbNodes)

	for i := 0; i < nbNodes; i++ {
		latencies := make(map[string]ConfirmedLatency)
		for k, v := range latencyMap {
			splitIndex := strings.Index(k, "-")
			n1 := k[:splitIndex]
			n2 := k[splitIndex+1:]

			if nodesToNumbers(n1) == i {
				latencies[n2] = ConfirmedLatency{time.Duration(v), nil, time.Now(), nil}
			}

			if nodesToNumbers(n2) == i {
				latencies[n1] = ConfirmedLatency{time.Duration(v), nil, time.Now(), nil}
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

func setLiarAndVictim(chain *Chain, liar string, victim string, latency time.Duration) {
	chain.Blocks[nodesToNumbers(liar)].Latencies[victim] = ConfirmedLatency{time.Duration(latency * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[nodesToNumbers(victim)].Latencies[liar] = ConfirmedLatency{time.Duration(latency * time.Nanosecond), nil, time.Now(), nil}

}

func deleteLatency(chain *Chain, node1 string, node2 string) {
	delete(chain.Blocks[nodesToNumbers(node1)].Latencies, node2)
	delete(chain.Blocks[nodesToNumbers(node2)].Latencies, node1)
}

func checkBlacklistWithRemovedLatencies(chain *Chain, nodes []sigAlg.PublicKey) string {

	str := ""
	recap := "\nRecap: \n"

	delta := time.Duration(0)
	threshold := UpperThreshold(len(chain.Blocks))

	originalBlacklist, _ := CreateBlacklist(chain, delta, false)

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
				newBlack, _ := CreateBlacklist(chain, delta, false)

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
		if !contains(b, a[i]) {
			return false
		}
	}

	return true
}

func contains(s []sigAlg.PublicKey, e sigAlg.PublicKey) bool {
	for _, a := range s {
		if reflect.DeepEqual(a, e) {
			return true
		}
	}
	return false
}
