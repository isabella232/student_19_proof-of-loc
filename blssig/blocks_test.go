package proofofloc

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"testing"
)

const blocksName = "testBlocks"

var tSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestNewChain(t *testing.T) {

	newChain := NewChain(tSuite)
	require.NotNil(t, newChain, "Could not create new chain")

}

func TestNewBlock(t *testing.T) {

	log.SetDebugVisible(1)

	var err error

	local := onet.NewTCPTest(tSuite)
	//_, roster, _ := local.GenTree(3, false)
	newNode1 := local.GenServers(1)
	newNode2 := local.GenServers(1)
	defer local.CloseAll()

	chain := NewChain(tSuite)

	newblock1, err := NewBlock(newNode1[0].ServerIdentity, chain)

	require.NoError(t, err)
	require.NotNil(t, newblock1, "Could not create new block")
	require.Zero(t, newblock1.nbReplies, "Should not have any latencies, as this is first block")
	require.Equal(t, newblock1.nbReplies, len(newblock1.Latencies), "nb replies not equal to number of latencies")

	require.Equal(t, 1, chain.nbBlocks, "Should have one block after first added")

	newblock2, err := NewBlock(newNode2[0].ServerIdentity, chain)

	require.NoError(t, err)
	require.NotNil(t, newblock2, "Could not create new block")
	require.Equal(t, 1, newblock2.nbReplies, "Should have 1 latency, as this is second block")
	require.Equal(t, newblock2.nbReplies, len(newblock2.Latencies), "nb replies not equal to number of latencies")
	require.NotEmpty(t, newblock2.Latencies, "Should have at least 1 latency")

	require.Equal(t, 2, chain.nbBlocks, "Should have two block after second added")

}
