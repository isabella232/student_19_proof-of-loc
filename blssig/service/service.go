// Package service implements a BLSCoSi service for which clients can connect to and then sign messages.
package service

import (
	"crypto/sha256"
	"errors"
	"github.com/dedis/student_19_proof-of-loc/blssig/proofofloc"
	"github.com/dedis/student_19_proof-of-loc/blssig/protocol"
	uuid "github.com/satori/go.uuid"
	"go.dedis.ch/cothority/v3/messaging"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
	bbolt "go.etcd.io/bbolt"
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
func NewChain(suite *pairing.SuiteBn256, Roster *onet.Roster) *proofofloc.Chain {
	chain := &proofofloc.Chain{Suite: suite, Roster: Roster, Blocks: make([]*proofofloc.Block, 0), BucketName: []byte("proofoflocBlocks")}

	return chain
}

//StoreBlock adds a block to a chain
func (s *BLSCoSiService) StoreBlock(request *StoreBlockRequest) (*StoreBlockResponse, error) {
	//do some work
	work(request.Block)

	//value is byte encoding of block
	value, err := protobuf.Encode(request.Block)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write(value)

	//key is the hash of the block
	key := h.Sum([]byte{})

	//Add block to chain
	db, bucket := s.GetAdditionalBucket([]byte(request.Chain.BucketName))

	db.Update(func(tx *bbolt.Tx) error {
		tx.Bucket(bucket).Put(key, value)
		return nil
	})

	//chain.Roster.Concat(block.id)
	request.Chain.Blocks = append(request.Chain.Blocks, request.Block)

	return &StoreBlockResponse{true}, nil
}

func work(block *proofofloc.Block) {

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
func ping(b *proofofloc.Block, dest *proofofloc.Block, suite *pairing.SuiteBn256) {

	//get random time delay between 20 and 300 ms - for now, just return this ----------------------------------------
	randomDelay := time.Duration((rand.Intn(300-20) + 20)) * time.Millisecond

	b.Latencies[dest.ID] = randomDelay
	b.NbReplies++
	// -----------------------------------------------------------------------------------

	conn, err := network.NewTCPConn(dest.ID.Address, suite)

	if err != nil {
		log.Error(err, "Couldn't create new TCP connection:")
		return
	}

	nonce := proofofloc.Nonce(rand.Int())

	b.Nonces[dest.ID] = nonce

	_, err1 := conn.Send(proofofloc.PingMsg{ID: b.ID, Nonce: nonce, IsReply: false, StartingTime: time.Now()})

	if err1 != nil {
		log.Error(err, "Couldn't send ping message:")
		return
	}

	conn.Close()

}

//pingListen listens for pings and pongs from other validators and handles them accordingly
func pingListen(b *proofofloc.Block, c network.Conn) {

	env, err := c.Receive()

	if err != nil {
		log.Error(err, "Couldn't send receive message from connection:")
		return
	}

	//Filter for the two types of messages we care about
	Msg, isPing := env.Msg.(proofofloc.PingMsg)

	// Case 1: someone pings ups -> reply with pong and control values
	if isPing {
		if !Msg.IsReply {
			c.Send(proofofloc.PingMsg{ID: b.ID, Nonce: Msg.Nonce, IsReply: true, StartingTime: Msg.StartingTime})
		} else {
			//Case 2: someone replies to our ping -> check return time
			if Msg.IsReply && b.Nonces[Msg.ID] == Msg.Nonce {

				latency := time.Since(Msg.StartingTime)
				b.Latencies[Msg.ID] = latency
				b.NbReplies++

			}
		}

		c.Close()

	}
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
