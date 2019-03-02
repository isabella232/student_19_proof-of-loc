package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/kyber/v3/sign/bls"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

var tSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}

//Note: this test arbitrarily passes or fails -> needs to be adapted
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

//Note: this test arbitrarily passes or fails -> needs to be adapted
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
