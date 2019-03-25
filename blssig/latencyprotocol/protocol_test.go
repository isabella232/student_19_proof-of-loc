package latencyprotocol

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/pairing"
	//"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"testing"
)

var tSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestNewNodeCreation(t *testing.T) {

	N := 3
	x := 2

	chain := initChain(N, x, accurate)

	log.LLvl1("Calling NewNode")
	_, finish, err := NewNode(chain.Blocks[0].ID.ServerID, tSuite, chain)

	log.LLvl1("Made new node")

	*finish <- true

	require.NoError(t, err)

}
