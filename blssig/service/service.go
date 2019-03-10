// Package service implements a BLSCoSi service for which clients can connect to and then sign messages.
package service

import (
	"errors"
	"github.com/dedis/student_19_proof-of-loc/blssig/protocol"
	uuid "github.com/satori/go.uuid"
	"go.dedis.ch/cothority/v3/messaging"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"math/rand"
	"time"
)

// This file contains all the code to run a BLSCoSi service. It is used to reply to
// client request for signing something using BLSCoSiService.
// This is an updated version of the CoSi Service (dedis/cothority/cosi/service), which only does simple signing

// ServiceName is the name to refer to BLSCoSiService
const ServiceName = "BLSCoSiService"

const protoName = "blscosiproto"

const nbPingsNeeded = 5

var serviceID onet.ServiceID

func init() {
	var err error
	serviceID, err = onet.RegisterNewService(ServiceName, newBLSCoSiService)
	log.ErrFatal(err)
	onet.GlobalProtocolRegister(protoName, protocol.NewDefaultProtocol)
	network.RegisterMessage(&SignatureRequest{})
	network.RegisterMessage(&SignatureResponse{})
	network.RegisterMessage(&PropagationFunction{})
}

// BLSCoSiService is the service that handles collective signing operations
type BLSCoSiService struct {
	*onet.ServiceProcessor
	propagationFunction messaging.PropagationFunc
	propagatedSignature []byte
}

//NewChain builds a new chain
func NewChain(suite *pairing.SuiteBn256) *Chain {
	return &Chain{suite, &onet.Roster{}, make([]*Block, 0)}
}

// SignatureRequest treats external requests to this service.
func (s *BLSCoSiService) SignatureRequest(req *SignatureRequest) (*SignatureResponse, error) {
	if req.Roster.ID.IsNil() {
		req.Roster.ID = onet.RosterID(uuid.NewV4())
	}

	_, root := req.Roster.Search(s.ServerIdentity().ID)
	if root == nil {
		return nil, errors.New("Couldn't find a serverIdentity in Roster")
	}

	tree := req.Roster.GenerateNaryTreeWithRoot(2, root)
	pi, err := s.CreateProtocol(protoName, tree)
	if err != nil {
		return nil, errors.New("Couldn't make new protocol: " + err.Error())
	}

	//Set message and start signing
	protocolInstance := pi.(*protocol.SimpleBLSCoSi)
	protocolInstance.Message = req.Message

	log.Lvl3("BLSCosi Service starting up root protocol")

	if err = pi.Start(); err != nil {
		return nil, err
	}

	//Get signature
	sig := <-protocolInstance.FinalSignature

	// We propagate the signature to all nodes
	err = s.startPropagation(s.propagationFunction, req.Roster, &PropagationFunction{sig})
	if err != nil {
		log.Error(err, "Couldn't propagate signature:")
		return nil, err
	}

	prop := s.propagatedSignature

	return &SignatureResponse{sig, prop}, nil

}

/*Ping allows a validator node to ping another node

The ping function is, for now, a random delay between 20 ms and 300 ms.

When node a pings node b, node a sends a message “ping” to node b (using onet) and node b replies with “pong” within a random delay time
*/
func (b *Block) Ping(dest *Block, suite *pairing.SuiteBn256) {

	//get random time delay between 20 and 300 ms - for now, just return this ----------------------------------------
	randomDelay := time.Duration((rand.Intn(300-20) + 20)) * time.Millisecond

	b.Latencies[dest.id] = randomDelay
	b.nbReplies++
	// -----------------------------------------------------------------------------------

	conn, err := network.NewTCPConn(dest.id.Address, suite)

	if err != nil {
		log.Error(err, "Couldn't create new TCP connection:")
		return
	}

	nonce := nonce(rand.Int())

	b.nonces[dest.id] = nonce

	_, err1 := conn.Send(PingMsg{b.id, nonce, false, time.Now()})

	if err1 != nil {
		log.Error(err, "Couldn't send ping message:")
		return
	}

	conn.Close()

}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func newBLSCoSiService(c *onet.Context) (onet.Service, error) {
	s := &BLSCoSiService{
		ServiceProcessor: onet.NewServiceProcessor(c),
	}
	err := s.RegisterHandler(s.SignatureRequest)
	if err != nil {
		log.Error(err, "Couldn't register handler:")
		return nil, err
	}

	s.propagationFunction, err = messaging.NewPropagationFunc(c, "propagateBLSCoSiSignature", s.propagateFuncHandler, -1)
	if err != nil {
		log.Error(err, "Couldn't create propagation function:")
		return nil, err
	}

	return s, nil
}

//startPropagation propagates the final signature to all the other nodes
func (s *BLSCoSiService) startPropagation(propagate messaging.PropagationFunc, ro *onet.Roster, msg network.Message) error {

	replies, err := propagate(ro, msg, 10*time.Second)
	if err != nil {
		log.Error(err, "Couldn't propagate signature:")
		return err
	}

	if replies != len(ro.List) {
		log.Lvl1(s.ServerIdentity(), "Only got", replies, "out of", len(ro.List))
	}

	return nil
}

// propagateForwardLinkHandler will update the propagated Signature with the latest one given to root node
func (s *BLSCoSiService) propagateFuncHandler(msg network.Message) error {
	s.propagatedSignature = msg.(*PropagationFunction).Signature
	return nil
}
