package latencyprotocol

import (
	//"github.com/stretchr/testify/require"
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

	N := 1
	x := 1
	// generate 3 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, el, _ := local.GenTree(1, false)

	defer local.CloseAll()

	chain := initChain(N, x, accurate)

	_, err := NewNode(el.List[0], tSuite, chain)

	log.LLvl1(err)

	//require.NoError(t, err)

}
