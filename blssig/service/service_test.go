package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/kyber/v3/sign/bls"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

const blocksName = "testBlocks"

var tSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestServiceBLSCosi(t *testing.T) {

	var err error

	local := onet.NewTCPTest(tSuite)
	// generate 3 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	hosts, el, _ := local.GenTree(3, false)
	defer local.CloseAll()

	aggregatePublicKey := bls.AggregatePublicKeys(tSuite, el.Publics()...)

	services := local.GetServices(hosts, serviceID)

	// Send a request to the service to all hosts
	msg := []byte("hello blscosi service")
	serviceReq := &SignatureRequest{
		Roster:  el,
		Message: msg,
	}

	log.Lvl2("Sending request to service...")
	s := services[0].(*BLSCoSiService)
	reply, err := s.SignatureRequest(serviceReq)
	require.Nil(t, err, "Couldn't send")
	require.NotEmpty(t, reply.Signature, "No signature")

	err = bls.Verify(tSuite, aggregatePublicKey, msg, reply.Signature)
	require.Nil(t, err, "Signature incorrect")

	err = bls.Verify(tSuite, aggregatePublicKey, msg, reply.Propagated)
	require.Nil(t, err, "Propagated incorrect")

}

func TestApi(t *testing.T) {

	var err error

	//log.SetDebugVisible(1)
	local := onet.NewTCPTest(tSuite)
	// generate 5 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist

	_, el, _ := local.GenTree(5, false)
	defer local.CloseAll()

	aggregatePublicKey := bls.AggregatePublicKeys(tSuite, el.Publics()...)

	// Send a request to the service
	client := NewClient()
	msg := []byte("hello blscosi service")

	el1 := &onet.Roster{}
	_, err = client.SignatureRequest(el1, msg)

	require.NotNil(t, err)
	// Create a roster with a missing aggregate and ID.
	el2 := &onet.Roster{List: el.List}

	res, err := client.SignatureRequest(el2, msg)

	require.Nil(t, err, "Couldn't send")
	require.NotNil(t, res, "No response")
	require.NotEmpty(t, res.Signature, "No response signature")

	err = bls.Verify(tSuite, aggregatePublicKey, msg, res.Signature)
	require.Nil(t, err, "Signature incorrect")

	err = bls.Verify(tSuite, aggregatePublicKey, msg, res.Propagated)
	require.Nil(t, err, "Propagation incorrect")
}

func TestNewChain(t *testing.T) {

	newChain := NewChain(tSuite, &onet.Roster{})
	require.NotNil(t, newChain, "Could not create new chain")

}

func TestNewBlock(t *testing.T) {

	log.SetDebugVisible(1)

	//var err error

	client := NewClient()

	local := onet.NewTCPTest(tSuite)

	_, el, _ := local.GenTree(3, false)

	newNode1 := local.GenServers(1)
	//newNode2 := local.GenServers(1)
	defer local.CloseAll()

	chain := NewChain(tSuite, el)

	newblock1, err := client.ProposeNewBlock(newNode1[0].ServerIdentity, chain)

	require.NoError(t, err)
	require.NotNil(t, newblock1, "Could not create new block")
	/*require.Zero(t, newblock1.nbReplies, "Should not have any latencies, as this is first block")
	require.Equal(t, newblock1.nbReplies, len(newblock1.Latencies), "nb replies not equal to number of latencies")

	require.Equal(t, 1, len(chain.blocks), "Should have one block after first added")

	newblock2, err := ProposeNewBlock(newNode2[0].ServerIdentity, chain)

	require.NoError(t, err)
	require.NotNil(t, newblock2, "Could not create new block")
	require.Equal(t, 1, newblock2.nbReplies, "Should have 1 latency, as this is second block")
	require.Equal(t, newblock2.nbReplies, len(newblock2.Latencies), "nb replies not equal to number of latencies")
	require.NotEmpty(t, newblock2.Latencies, "Should have at least 1 latency")

	require.Equal(t, 2, len(chain.blocks), "Should have two block after second added")*/

}
