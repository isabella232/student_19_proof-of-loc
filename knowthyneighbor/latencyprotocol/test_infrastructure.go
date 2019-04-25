package latencyprotocol

import (
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	sigAlg "golang.org/x/crypto/ed25519"
	"math/rand"
	"time"
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
