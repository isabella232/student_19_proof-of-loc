package proofofloc

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	//"math/rand"
	"testing"
	"time"
)

var tSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}

/*Test it initially by assuming
all nodes are honest, each node adds in the blockchain x distances from itself to other x nodes,
where these x nodes are randomly chosen. You can assume for now that there’s a publicly known source
of randomness that nodes use. Check the results by varying the number x and the total number of nodes N.*/
func TestApproximateDistanceCompleteInformation(t *testing.T) {

	N := 3
	x := 2

	local := onet.NewTCPTest(tSuite)
	// generate 3 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, el, _ := local.GenTree(N, false)
	defer local.CloseAll()

	chain := Chain{[]*Block{}, []byte("testBucket")}

	for i := 0; i < N; i++ {
		latencies := make(map[*network.ServerIdentity]time.Duration)
		id := el.List[i]
		log.Print(id)
		for j := 0; j < x; j++ {
			if i != j {
				//latencies[el.List[j]] = time.Duration((rand.Intn(300-20) + 20)) * time.Millisecond
				latencies[el.List[j]] = time.Duration(10*(i+j)) * time.Millisecond
			}
		}
		chain.Blocks = append(chain.Blocks, &Block{id, latencies})
	}

	d12, err := chain.Blocks[0].ApproximateDistance(chain.Blocks[1], chain.Blocks[2], 10)

	require.Nil(t, err, "Error")
	require.Equal(t, d12, 10*(1+2)*time.Millisecond)

	d02, err := chain.Blocks[1].ApproximateDistance(chain.Blocks[0], chain.Blocks[2], 10)

	require.Nil(t, err, "Error")
	require.Equal(t, d02, 10*(2)*time.Millisecond)

	d01, err := chain.Blocks[2].ApproximateDistance(chain.Blocks[0], chain.Blocks[1], 10)

	require.Nil(t, err, "Error")
	require.Equal(t, d01, 10*(1)*time.Millisecond)

}
