package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

var tSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestServiceBLSCosi(t *testing.T) {

	log.SetDebugVisible(1)

	local := onet.NewTCPTest(tSuite)
	// generate 5 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, el, _ := local.GenTree(2, false)
	defer local.CloseAll()

	// Send a request to the service to all hosts
	client := NewClient()
	msg := []byte("hello blscosi service")
	serviceReq := &SignatureRequest{
		Roster:  el,
		Message: msg,
	}

	for _, dst := range el.List {
		reply := &SignatureResponse{}
		log.Lvl2("Sending request to service...")
		err := client.SendProtobuf(dst, serviceReq, reply)
		require.Nil(t, err, "Couldn't send")
		require.NotNil(t, reply.Signature, "No response")
		//log.Print("Signature: ")
		//log.Print(reply.Signature)
	}
}

func TestCreateAggregate(t *testing.T) {

	//log.SetDebugVisible(1)
	local := onet.NewTCPTest(tSuite)
	// generate 5 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist

	_, el, _ := local.GenTree(5, false)
	defer local.CloseAll()

	// Send a request to the service
	client := NewClient()
	msg := []byte("hello cosi service")
	log.Lvl1("Sending request to service...")

	el1 := &onet.Roster{}
	_, err := client.SignatureRequest(el1, msg)

	require.NotNil(t, err)
	// Create a roster with a missing aggregate and ID.
	el2 := &onet.Roster{List: el.List}

	res, err := client.SignatureRequest(el2, msg)

	require.Nil(t, err, "Couldn't send")
	require.NotNil(t, res, "No response")
	require.NotNil(t, res.Signature, "No response signature")
	require.NotNil(t, res.Propagated, "No propagated signature")
}
