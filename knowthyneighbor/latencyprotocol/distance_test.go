package latencyprotocol

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	sigAlg "golang.org/x/crypto/ed25519"
	"math/rand"
	"reflect"
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

func initChain(N int, x int, src sourceType) (*Chain, []*NodeID) {
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
					latencies[string(nodeIDs[j].PublicKey)] = ConfirmedLatency{
						time.Duration(10 * (i + j + 1 + rand.Intn(5))),
						nil,
						time.Now(),
						nil,
					}
				case inaccurate:
					latencies[string(nodeIDs[j].PublicKey)] = ConfirmedLatency{
						time.Duration(((i * 10000) + j + 1)),
						nil,
						time.Now(),
						nil,
					}

				}

			}
		}
		chain.Blocks = append(chain.Blocks, &Block{nodeIDs[i], latencies})
	}

	return &chain, nodeIDs

}

func TestApproximateDistanceAllInformation(t *testing.T) {

	N := 3
	x := 2

	chain, _ := initChain(N, x, accurate)

	_, exists := chain.Blocks[0].Latencies[string(chain.Blocks[1].ID.PublicKey)]

	log.Print(exists)

	d12, isValid12, err := chain.Blocks[0].ApproximateDistance(chain.Blocks[1], chain.Blocks[2], 10)

	require.Nil(t, err, "Error")
	require.Equal(t, d12, time.Duration(10*(1+2+1)))
	require.True(t, isValid12)

	d02, isValid02, err := chain.Blocks[1].ApproximateDistance(chain.Blocks[0], chain.Blocks[2], 10)

	require.Nil(t, err, "Error")
	require.Equal(t, d02, time.Duration(10*(2+1)))
	require.True(t, isValid02)

	d01, isValid01, err := chain.Blocks[2].ApproximateDistance(chain.Blocks[0], chain.Blocks[1], 10)

	require.Nil(t, err, "Error")
	require.Equal(t, d01, time.Duration(10*(1+1)))
	require.True(t, isValid01)

}

func TestApproximateDistanceInaccurateInformation(t *testing.T) {

	N := 6
	x := 4

	chain, _ := initChain(N, x, inaccurate)

	_, isValid, err := chain.Blocks[0].ApproximateDistance(chain.Blocks[1], chain.Blocks[2], 0)

	require.NotNil(t, err, "Inaccuracy error should have been reported")
	require.False(t, isValid)

}

func TestApproximateDistanceIncompleteInformation(t *testing.T) {

	/* Test Environment:

	N1---(d01 + d10/2)----N0----d02----N2

	N1-N2 unknown by any nodes -> pythagoras
	N0 - N2 only given by one node -> not trustworthy


	*/

	N := 3
	x := 1

	expectedD01 := time.Duration(10003 / 2)
	expectedD02 := time.Duration(((2 * 10000) + 1))
	expectedD12 := Pythagoras(expectedD01, expectedD02)

	chain, _ := initChain(N, x, inaccurate)

	d01, isValid01, err := chain.Blocks[2].ApproximateDistance(chain.Blocks[0], chain.Blocks[1], 10000)

	require.Nil(t, err, "Error")
	require.Equal(t, d01, expectedD01)
	require.True(t, isValid01)

	_, isValid02, err := chain.Blocks[1].ApproximateDistance(chain.Blocks[0], chain.Blocks[2], 10000)

	require.NotNil(t, err)
	require.False(t, isValid02)

	d12, isValid12, err := chain.Blocks[0].ApproximateDistance(chain.Blocks[1], chain.Blocks[2], 10000)

	require.Nil(t, err, "Error")
	require.Equal(t, d12, expectedD12)
	require.True(t, isValid12)

}

func TestApproximateDistanceMissingInformation(t *testing.T) {

	N := 5
	x := 1

	chain, _ := initChain(N, x, accurate)

	_, isValid, err := chain.Blocks[2].ApproximateDistance(chain.Blocks[3], chain.Blocks[4], 0)

	require.NotNil(t, err, "Should not have sufficient information to approximate distance")
	require.False(t, isValid)

}

func TestBlacklistOnAccurateChainEmpty(t *testing.T) {

	N := 4
	x := 4
	d := 1 * time.Nanosecond
	suspicionThreshold := 0

	chain, nodeIDs := initChain(N, x, accurate)

	for _, NodeID := range nodeIDs {
		node := Node{
			ID:                      NodeID,
			SendingAddress:          "address",
			PrivateKey:              nil,
			LatenciesInConstruction: nil,
			BlockSkeleton:           nil,
			NbLatenciesRefreshed:    0,
			IncomingMessageChannel:  nil,
			BlockChannel:            nil,
		}

		blacklist, err := node.CreateBlacklist(chain, d, suspicionThreshold)

		require.NoError(t, err)
		require.Zero(t, len(blacklist), "Blacklist should be empty")

	}
}

func TestAllBlacklistsOnInaccurateChainIdentical(t *testing.T) {

	N := 4
	x := 4
	d := 1 * time.Nanosecond
	suspicionThreshold := 0

	blacklists := make([][]sigAlg.PublicKey, N)

	chain, nodeIDs := initChain(N, x, random)

	for index, NodeID := range nodeIDs {
		node := Node{
			ID:                      NodeID,
			SendingAddress:          "address",
			PrivateKey:              nil,
			LatenciesInConstruction: nil,
			BlockSkeleton:           nil,
			NbLatenciesRefreshed:    0,
			IncomingMessageChannel:  nil,
			BlockChannel:            nil,
		}

		blacklist, err := node.CreateBlacklist(chain, d, suspicionThreshold)

		require.NoError(t, err)
		require.NotZero(t, len(blacklist))
		blacklists[index] = blacklist

	}

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklistsEquivalent(blacklist, blacklists[0]))
	}

}

func TestExactlyOneLiarBlacklistedSmall(t *testing.T) {

	// A <-> D does not make sense, not enough info to know who is evil

	N := 4
	d := 1 * time.Nanosecond
	suspicionThreshold := 1

	A := &Block{
		ID: &NodeID{
			ServerID:  nil,
			PublicKey: sigAlg.PublicKey("A"),
		},
		Latencies: map[string]ConfirmedLatency{
			"B": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"C": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
			"D": ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil},
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
			"D": ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil},
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
			"A": ConfirmedLatency{time.Duration(25 * time.Nanosecond), nil, time.Now(), nil},
			"B": ConfirmedLatency{time.Duration(12 * time.Nanosecond), nil, time.Now(), nil},
			"C": ConfirmedLatency{time.Duration(10 * time.Nanosecond), nil, time.Now(), nil},
		},
	}

	chain := &Chain{
		Blocks:     []*Block{A, B, C, D},
		BucketName: []byte("TestBucket"),
	}

	blacklists := make([][]sigAlg.PublicKey, N)

	for index, key := range []sigAlg.PublicKey{
		sigAlg.PublicKey("A"),
		sigAlg.PublicKey("B"),
		sigAlg.PublicKey("C"),
		sigAlg.PublicKey("D")} {
		node := Node{
			ID:                      &NodeID{nil, key},
			SendingAddress:          "address",
			PrivateKey:              nil,
			LatenciesInConstruction: nil,
			BlockSkeleton:           nil,
			NbLatenciesRefreshed:    0,
			IncomingMessageChannel:  nil,
			BlockChannel:            nil,
		}

		blacklist, err := node.CreateBlacklist(chain, d, suspicionThreshold)

		require.NoError(t, err)
		require.NotZero(t, len(blacklist))
		blacklists[index] = blacklist

	}

	firstBlacklist := blacklists[0]

	require.Equal(t, 2, len(firstBlacklist))
	require.Contains(t, firstBlacklist, sigAlg.PublicKey([]byte("A")), sigAlg.PublicKey([]byte("D")))

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklistsEquivalent(blacklist, firstBlacklist))
	}

}

func TestExactlyOneLiarBlacklistedLarge(t *testing.T) {

	N := 5
	d := 1 * time.Nanosecond
	suspicionThreshold := 2

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

	blacklists := make([][]sigAlg.PublicKey, N)

	for index, key := range []sigAlg.PublicKey{
		sigAlg.PublicKey("A"),
		sigAlg.PublicKey("B"),
		sigAlg.PublicKey("C"),
		sigAlg.PublicKey("D"),
		sigAlg.PublicKey("E")} {
		node := Node{
			ID:                      &NodeID{nil, key},
			SendingAddress:          "address",
			PrivateKey:              nil,
			LatenciesInConstruction: nil,
			BlockSkeleton:           nil,
			NbLatenciesRefreshed:    0,
			IncomingMessageChannel:  nil,
			BlockChannel:            nil,
		}

		blacklist, err := node.CreateBlacklist(chain, d, suspicionThreshold)

		require.NoError(t, err)
		require.NotZero(t, len(blacklist))
		blacklists[index] = blacklist

	}

	firstBlacklist := blacklists[0]

	require.Equal(t, 1, len(firstBlacklist))
	require.Contains(t, firstBlacklist, sigAlg.PublicKey([]byte("C")))

	for _, blacklist := range blacklists[1:] {
		require.True(t, blacklistsEquivalent(blacklist, firstBlacklist))
	}

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
