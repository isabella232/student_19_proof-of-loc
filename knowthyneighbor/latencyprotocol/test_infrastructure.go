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

func fourNodeChain() (*Chain, []sigAlg.PublicKey) {

	A := &Block{
		ID: &NodeID{
			ServerID:  nil,
			PublicKey: sigAlg.PublicKey("A"),
		},
		Latencies: map[string]ConfirmedLatency{
			"B": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"C": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"D": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
		},
	}

	B := &Block{
		ID: &NodeID{
			ServerID:  nil,
			PublicKey: sigAlg.PublicKey("B"),
		},
		Latencies: map[string]ConfirmedLatency{
			"A": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"C": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"D": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
		},
	}

	C := &Block{
		ID: &NodeID{
			ServerID:  nil,
			PublicKey: sigAlg.PublicKey("C"),
		},
		Latencies: map[string]ConfirmedLatency{
			"A": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"B": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"D": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
		},
	}

	D := &Block{
		ID: &NodeID{
			ServerID:  nil,
			PublicKey: sigAlg.PublicKey("D"),
		},
		Latencies: map[string]ConfirmedLatency{
			"A": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"B": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"C": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
		},
	}

	chain := &Chain{
		Blocks:     []*Block{A, B, C, D},
		BucketName: []byte("TestBucket"),
	}

	nodes := []sigAlg.PublicKey{
		sigAlg.PublicKey("A"),
		sigAlg.PublicKey("B"),
		sigAlg.PublicKey("C"),
		sigAlg.PublicKey("D")}

	return chain, nodes
}

func fiveNodeChain() (*Chain, []sigAlg.PublicKey) {
	A := &Block{
		ID: &NodeID{
			ServerID:  nil,
			PublicKey: sigAlg.PublicKey("A"),
		},
		Latencies: map[string]ConfirmedLatency{
			"B": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"C": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"D": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"E": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
		},
	}

	B := &Block{
		ID: &NodeID{
			ServerID:  nil,
			PublicKey: sigAlg.PublicKey("B"),
		},
		Latencies: map[string]ConfirmedLatency{
			"A": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"C": ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil},
			"D": ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil},
			"E": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
		},
	}

	C := &Block{
		ID: &NodeID{
			ServerID:  nil,
			PublicKey: sigAlg.PublicKey("C"),
		},
		Latencies: map[string]ConfirmedLatency{
			"A": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"B": ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil},
			"D": ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil},
			"E": ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil},
		},
	}

	D := &Block{
		ID: &NodeID{
			ServerID:  nil,
			PublicKey: sigAlg.PublicKey("D"),
		},
		Latencies: map[string]ConfirmedLatency{
			"A": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"B": ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil},
			"C": ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil},
			"E": ConfirmedLatency{time.Duration(20 * time.Nanosecond), nil, time.Now(), nil},
		},
	}

	E := &Block{
		ID: &NodeID{
			ServerID:  nil,
			PublicKey: sigAlg.PublicKey("E"),
		},
		Latencies: map[string]ConfirmedLatency{
			"A": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"B": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"C": ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil},
			"D": ConfirmedLatency{time.Duration(20 * time.Nanosecond), nil, time.Now(), nil},
		},
	}

	chain := &Chain{
		Blocks:     []*Block{A, B, C, D, E},
		BucketName: []byte("TestBucket"),
	}

	nodes := []sigAlg.PublicKey{
		sigAlg.PublicKey("A"),
		sigAlg.PublicKey("B"),
		sigAlg.PublicKey("C"),
		sigAlg.PublicKey("D"),
		sigAlg.PublicKey("E")}

	return chain, nodes
}

func setLiarAndVictim(chain *Chain, liar string, victim string, latency int) {

	blockNbLiar := 0
	switch liar {
	case "B":
		blockNbLiar = 1
	case "C":
		blockNbLiar = 2
	case "D":
		blockNbLiar = 3
	case "E":
		blockNbLiar = 4
	}

	blockNbVictim := 0
	switch victim {
	case "B":
		blockNbVictim = 1
	case "C":
		blockNbVictim = 2
	case "D":
		blockNbVictim = 3
	case "E":
		blockNbVictim = 4
	}

	chain.Blocks[blockNbLiar].Latencies[victim] = ConfirmedLatency{time.Duration(time.Duration(latency) * time.Nanosecond), nil, time.Now(), nil}
	chain.Blocks[blockNbVictim].Latencies[liar] = ConfirmedLatency{time.Duration(time.Duration(latency) * time.Nanosecond), nil, time.Now(), nil}

}
