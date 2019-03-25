package latencyprotocol

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"testing"
)

var tSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestNewNodeCreation(t *testing.T) {

	local := onet.NewTCPTest(tSuite)
	// generate 3 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, el, _ := local.GenTree(1, false)
	defer local.CloseAll()

	log.LLvl1("Calling NewNode")
	newNode, finish, err := NewNode(el.List[0], tSuite, 2)

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
	_, el, _ := local.GenTree(2, false)
	defer local.CloseAll()

	chain := &Chain{make([]*Block, 1), []byte("testBucket")}

	newNode1, finish1, err := NewNode(el.List[0], tSuite, 0)
	require.NoError(t, err)

	chain.Blocks[0] = &Block{newNode1.ID, make(map[string]ConfirmedLatency, 0)}

	newNode2, finish2, err := NewNode(el.List[1], tSuite, 1)

	require.NoError(t, err)

	newNode2.AddBlock(chain)

	block := <-newNode2.BlockChannel

	require.NotNil(t, block)

	finish1 <- true
	finish2 <- true

}
