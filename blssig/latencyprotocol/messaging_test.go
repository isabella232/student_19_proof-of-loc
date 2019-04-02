package latencyprotocol

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3"
	"testing"
	"time"
)

const benchmarkDelta = 10 * time.Second

func constructBlocks() ([]*Node, *Chain, error) {

	local := onet.NewTCPTest(tSuite)
	// generate 3 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, el, _ := local.GenTree(4, false)
	defer local.CloseAll()

	chain := &Chain{make([]*Block, 3), []byte("testBucket")}

	newNode1, finish1, err := NewNode(el.List[0], el.List[1].Address, tSuite, 1)
	if err != nil {
		return nil, nil, err
	}

	chain.Blocks[0] = &Block{newNode1.ID, make(map[string]ConfirmedLatency, 0)}

	newNode2, finish2, err := NewNode(el.List[2], el.List[3].Address, tSuite, 1)
	if err != nil {
		return nil, nil, err
	}

	newNode2.AddBlock(chain)

	block1 := <-newNode1.BlockChannel
	block2 := <-newNode2.BlockChannel

	finish1 <- true
	finish2 <- true

	chain.Blocks[1] = &block1
	chain.Blocks[2] = &block2

	nodes := make([]*Node, 2)
	nodes[0] = newNode1
	nodes[1] = newNode2

	return nodes, chain, nil

}

func TestCompareLatenciesToPings(t *testing.T) {

	nodes, chain, err := constructBlocks()

	require.NoError(t, err)

	latency0, err := Ping(nodes[0].ID.ServerID.String(), nodes[0].SendingAddress.String())
	require.NoError(t, err)
	latency1, err := Ping(nodes[1].ID.ServerID.String(), nodes[1].SendingAddress.String())
	require.NoError(t, err)

	expectedLat0 := chain.Blocks[0].Latencies[string(nodes[1].ID.PublicKey)].Latency
	expectedLat1 := chain.Blocks[1].Latencies[string(nodes[0].ID.PublicKey)].Latency

	require.True(t, (expectedLat0-latency0) < benchmarkDelta)
	require.True(t, (expectedLat1-latency1) < benchmarkDelta)
}
