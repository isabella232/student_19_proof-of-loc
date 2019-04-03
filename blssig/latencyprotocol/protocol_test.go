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
	// generate 3 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, el, _ := local.GenTree(2, false)
	defer local.CloseAll()

	log.LLvl1("Calling NewNode")
	newNode, finish, err := NewNode(el.List[0], el.List[1].Address, tSuite, 2)

	finish <- true

	log.LLvl1("Made new node")

	require.NoError(t, err)
	require.NotNil(t, newNode)
	require.Equal(t, newNode.ID.ServerID, el.List[0])

}

func TestAddBlock(t *testing.T) {

	local := onet.NewTCPTest(tSuite)
	// generate 3 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, el, _ := local.GenTree(4, false)
	defer local.CloseAll()

	chain := &Chain{make([]*Block, 1), []byte("testBucket")}

	newNode1, finish1, err := NewNode(el.List[0], el.List[1].Address, tSuite, 1)
	require.NoError(t, err)

	chain.Blocks[0] = &Block{newNode1.ID, make(map[string]ConfirmedLatency, 0)}

	newNode2, finish2, err := NewNode(el.List[2], el.List[3].Address, tSuite, 1)

	require.NoError(t, err)

	newNode2.AddBlock(chain)

	block1 := <-newNode1.BlockChannel

	log.LLvl1("Channel 1 got its block")

	finish1 <- true

	block2 := <-newNode2.BlockChannel

	log.LLvl1("Channel 2 got its block")

	finish2 <- true

	require.NotNil(t, block1)
	require.NotNil(t, block2)

	require.Equal(t, newNode1.ID, block1.ID)
	require.Equal(t, newNode2.ID, block2.ID)

	require.Len(t, block1.Latencies, 1)
	require.Equal(t, 1, len(block2.Latencies))
	require.Contains(t, block1.Latencies, string(block2.ID.PublicKey))
	require.Contains(t, block2.Latencies, string(block1.ID.PublicKey))

	latency1 := block1.Latencies[string(block2.ID.PublicKey)].Latency
	latency2 := block2.Latencies[string(block1.ID.PublicKey)].Latency

	require.NotZero(t, latency1)
	require.NotZero(t, latency2)

	latencyDiff := latency1 - latency2

	require.True(t, (latencyDiff < 10*time.Millisecond))
	require.True(t, (latencyDiff > 0*time.Millisecond))

}
