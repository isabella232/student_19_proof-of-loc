package latencyprotocol

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	sigAlg "golang.org/x/crypto/ed25519"
	"math/rand"
	"testing"
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

func initChain(N int, x int, src sourceType) *Chain {
	local := onet.NewTCPTest(distanceSuite)
	// generate 3 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, el, _ := local.GenTree(N, false)
	defer local.CloseAll()

	nodeIDs := make([]*NodeID, N)
	privateKeys := make([]sigAlg.PrivateKey, N)

	for h := 0; h < N; h++ {
		pub, priv, err := sigAlg.GenerateKey(nil)
		if err != nil {
			return nil
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
						time.Now(),
						nil,
					}
				case accurate:
					latencies[string(nodeIDs[j].PublicKey)] =
						ConfirmedLatency{
							time.Duration(10 * (i + j + 1)),
							time.Now(),
							nil,
						}
				case variant:
					latencies[string(nodeIDs[j].PublicKey)] = ConfirmedLatency{
						time.Duration(10 * (i + j + 1 + rand.Intn(5))),
						time.Now(),
						nil,
					}
				case inaccurate:
					latencies[string(nodeIDs[j].PublicKey)] = ConfirmedLatency{
						time.Duration(((i * 10000) + j + 1)),
						time.Now(),
						nil,
					}

				}

			}
		}
		chain.Blocks = append(chain.Blocks, &Block{nodeIDs[i], latencies})
	}

	return &chain

}

func TestApproximateDistanceAllInformation(t *testing.T) {

	N := 3
	x := 2

	chain := initChain(N, x, accurate)

	_, exists := chain.Blocks[0].Latencies[string(chain.Blocks[1].ID.PublicKey)]

	log.Print(exists)

	d12, err := chain.Blocks[0].ApproximateDistance(chain.Blocks[1], chain.Blocks[2], 10)

	require.Nil(t, err, "Error")
	require.Equal(t, d12, time.Duration(10*(1+2+1)))

	d02, err := chain.Blocks[1].ApproximateDistance(chain.Blocks[0], chain.Blocks[2], 10)

	require.Nil(t, err, "Error")
	require.Equal(t, d02, time.Duration(10*(2+1)))

	d01, err := chain.Blocks[2].ApproximateDistance(chain.Blocks[0], chain.Blocks[1], 10)

	require.Nil(t, err, "Error")
	require.Equal(t, d01, time.Duration(10*(1+1)))

}

func TestApproximateDistanceInaccurateInformation(t *testing.T) {

	N := 6
	x := 4

	chain := initChain(N, x, inaccurate)

	_, err := chain.Blocks[0].ApproximateDistance(chain.Blocks[1], chain.Blocks[2], 0)

	require.NotNil(t, err, "Inaccuracy error should have been reported")

}

func TestApproximateDistanceIncompleteInformation(t *testing.T) {

	/* Test Environment:

	N1---(d01 + d10/2)----N0----d02----N2

	N1-N2 unknown by any nodes -> pythagoras


	*/

	N := 3
	x := 1

	expectedD01 := time.Duration(10003 / 2)
	expectedD02 := time.Duration(((2 * 10000) + 1))
	expectedD12 := Pythagoras(expectedD01, expectedD02)

	chain := initChain(N, x, inaccurate)

	d01, err := chain.Blocks[2].ApproximateDistance(chain.Blocks[0], chain.Blocks[1], 10000)

	require.Nil(t, err, "Error")
	require.Equal(t, d01, expectedD01)

	d02, err := chain.Blocks[1].ApproximateDistance(chain.Blocks[0], chain.Blocks[2], 10000)

	require.Nil(t, err, "Error")
	require.Equal(t, d02, expectedD02)

	d12, err := chain.Blocks[0].ApproximateDistance(chain.Blocks[1], chain.Blocks[2], 10000)

	require.Nil(t, err, "Error")
	require.Equal(t, d12, expectedD12)

}

func TestApproximateDistanceMissingInformation(t *testing.T) {

	N := 5
	x := 1

	chain := initChain(N, x, accurate)

	_, err := chain.Blocks[2].ApproximateDistance(chain.Blocks[3], chain.Blocks[4], 0)

	require.NotNil(t, err, "Should not have sufficient information to approximate distance")

}
