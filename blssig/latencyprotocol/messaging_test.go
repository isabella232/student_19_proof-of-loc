package latencyprotocol

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
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

	chain := &Chain{make([]*Block, 1), []byte("testBucket")}

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

	log.LLvl1("Channel 1 got its block")

	finish1 <- true

	block2 := <-newNode2.BlockChannel

	log.LLvl1("Channel 2 got its block")

	finish2 <- true

	log.LLvl1("Storing blocks")
	chain.Blocks = append(chain.Blocks, &block1)
	chain.Blocks = append(chain.Blocks, &block2)

	log.LLvl1("Storing nodes")
	nodes := make([]*Node, 2)
	nodes[0] = newNode1
	nodes[1] = newNode2

	return nodes, chain, nil

}

func TestCompareLatenciesToPings(t *testing.T) {

	log.LLvl1("Make chain")
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
