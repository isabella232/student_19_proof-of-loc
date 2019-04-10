package latencyprotocol

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"testing"
	"time"
)

var tSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestNewNodeCreation(t *testing.T) {

	local := onet.NewTCPTest(tSuite)
	local.Check = onet.CheckNone
	// generate 3 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, el, _ := local.GenTree(2, false)
	defer local.CloseAll()

	newNode, finish, wg, err := NewNode(el.List[0], el.List[1].Address, tSuite, 2)

	finish <- true
	wg.Wait()

	require.NoError(t, err)
	require.NotNil(t, newNode)
	require.Equal(t, newNode.ID.ServerID, el.List[0])

}

func TestAddBlock(t *testing.T) {

	local := onet.NewTCPTest(tSuite)
	local.Check = onet.CheckNone
	// generate 3 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, el, _ := local.GenTree(4, false)
	defer local.CloseAll()

	chain := &Chain{make([]*Block, 1), []byte("testBucket")}

	newNode1, finish1, wg1, err := NewNode(el.List[0], el.List[1].Address, tSuite, 1)
	require.NoError(t, err)

	chain.Blocks[0] = &Block{newNode1.ID, make(map[string]ConfirmedLatency, 0)}

	newNode2, finish2, wg2, err := NewNode(el.List[2], el.List[3].Address, tSuite, 1)

	require.NoError(t, err)

	newNode2.AddBlock(chain)

	block1 := <-newNode1.BlockChannel

	finish1 <- true
	wg1.Wait()

	block2 := <-newNode2.BlockChannel

	finish2 <- true
	wg2.Wait()

	require.NotNil(t, block1, "Nil block")
	require.NotNil(t, block2, "Nil block")

	require.Equal(t, newNode1.ID, block1.ID, "Wrong id")
	require.Equal(t, newNode2.ID, block2.ID, "Wrong id")

	require.Len(t, block1.Latencies, 1, "Wrong number of latencies")
	require.Equal(t, 1, len(block2.Latencies), "Wrong number of latencies")
	require.Contains(t, block1.Latencies, string(block2.ID.PublicKey), "latency missing")
	require.Contains(t, block2.Latencies, string(block1.ID.PublicKey), "latency missing")

	latency1 := block1.Latencies[string(block2.ID.PublicKey)].Latency
	latency2 := block2.Latencies[string(block1.ID.PublicKey)].Latency

	require.NotZero(t, latency1, "Zero latency")
	require.NotZero(t, latency2, "Zero latency")

	latencyDiff1 := latency1 - latency2
	latencyDiff2 := latency2 - latency1

	require.True(t, (latencyDiff1 < 10*time.Millisecond) || (latencyDiff2 < 10*time.Millisecond), "latency differencetoo long")

}
