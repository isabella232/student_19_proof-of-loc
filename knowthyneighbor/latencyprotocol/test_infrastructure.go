/*
functions to create test chains

*/

package latencyprotocol

import (
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"

	"math/rand"
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

func chain(N int, x int, src sourceType, nbLiars int, nbVictims int) (*Chain, []*NodeID) {
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

func chainWithCClusters(C int, clusterSizes []int, distance int) (*Chain, []*Chain, [][]string) {

	clusters := make([]*Chain, C)
	nodeLists := make([][]string, C)

	N := 0

	for i := 0; i < C; i++ {
		cluster, nodeList := chainWithOnlyConsistentLatencies(clusterSizes[i], N)
		clusters[i] = cluster
		nodeLists[i] = nodeList
		N += clusterSizes[i]
	}

	masterBlocks := make([]*Block, N)
	masterIndex := 0

	//Steps:
	//1: for each chain, add latency to blocks of all other chains (careful: latencies go 2 ways -> only give forward)
	//2: Connect all chains

	//1
	//for each chain
	for j := 0; j < C; j++ {
		cluster := clusters[j]

		//for each block in chain
		for b := 0; b < len(cluster.Blocks); b++ {
			block := cluster.Blocks[b]

			//for each other chain
			for nl := 0; nl < C; nl++ {
				if nl != j {
					nodes := nodeLists[nl]
					for n := 0; n < len(nodes); n++ {
						node := nodes[n]
						randAddition := rand.Intn(500)
						newLat := ConfirmedLatency{time.Duration(distance + randAddition), nil, time.Now(), nil}
						block.Latencies[node] = newLat
						clusters[nl].Blocks[n].Latencies[numbersToNodes(masterIndex)] = newLat
					}

				}
			}

			//Copy completed block to big graph
			masterBlocks[masterIndex] = block
			masterIndex++

		}
	}

	//2

	masterChain := Chain{masterBlocks, []byte("testBucketName")}

	return &masterChain, clusters, nodeLists

}

func numbersToNodes(n int) string {
	return "N" + strconv.Itoa(n)
}

func nodesToNumbers(node string) int {
	nb, _ := strconv.Atoi(string(node[1:]))
	return nb
}

func chainWithAllLatenciesSame(nbNodes int, latency int) (*Chain, []sigAlg.PublicKey) {
	blocks := make([]*Block, nbNodes)
	nodes := make([]sigAlg.PublicKey, nbNodes)

	for i := 0; i < nbNodes; i++ {
		latencies := make(map[string]ConfirmedLatency)

		for j := 0; j < nbNodes; j++ {
			if j != i {
				latencies[numbersToNodes(j)] = ConfirmedLatency{time.Duration(latency), nil, time.Now(), nil}
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
func chainWithOnlyConsistentLatencies(nbNodes int, startIndex int) (*Chain, []string) {

	blocks := make([]*Block, nbNodes)
	nodes := make([]string, nbNodes)

	for i := 0; i < nbNodes; i++ {
		latencies := make(map[string]ConfirmedLatency)

		for j := 0; j < nbNodes; j++ {
			if j > i {
				lat := rand.Intn(500) + 500
				latencies[numbersToNodes(j+startIndex)] = ConfirmedLatency{time.Duration(lat), nil, time.Now(), nil}
			} else {
				if j < i {
					latencies[numbersToNodes(j+startIndex)] = blocks[j].Latencies[numbersToNodes(i)]
				}
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

func setLiesToClusters(liarID int, consistentChain *Chain) *Chain {

	inconsistentChain := consistentChain.Copy()

	for i := 0; i < len(inconsistentChain.Blocks); i++ {
		block := inconsistentChain.Blocks[i]

		_, isPresent := block.Latencies[numbersToNodes(liarID)]
		if isPresent {

			//Normal range within cluster
			lat := rand.Intn(500) + 500
			inconsistentChain.Blocks[liarID].Latencies[numbersToNodes(i)] = ConfirmedLatency{time.Duration(lat), nil, time.Now(), nil}
			block.Latencies[numbersToNodes(liarID)] = ConfirmedLatency{time.Duration(lat), nil, time.Now(), nil}
		}
	}

	return inconsistentChain
}

func setMultipleLiesToClusters(nbLiars int, consistentChain *Chain) *Chain {

	inconsistentChain := consistentChain.Copy()

	for i := 0; i < len(inconsistentChain.Blocks); i++ {
		block := inconsistentChain.Blocks[i]

		for liarID := 0; liarID < nbLiars; liarID++ {

			_, isPresent := block.Latencies[numbersToNodes(liarID)]
			if isPresent {

				//Normal range within cluster
				lat := rand.Intn(500) + 500
				inconsistentChain.Blocks[liarID].Latencies[numbersToNodes(i)] = ConfirmedLatency{time.Duration(lat), nil, time.Now(), nil}
				block.Latencies[numbersToNodes(liarID)] = ConfirmedLatency{time.Duration(lat), nil, time.Now(), nil}
			}
		}
	}

	return inconsistentChain
}
