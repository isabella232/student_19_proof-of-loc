// Package service implements a BLSCoSi service for which clients can connect to and then sign messages.
package service

import (
	"crypto/sha256"
	"errors"
	"github.com/dedis/student_19_proof-of-loc/blssig/blscosiprotocol"
	"github.com/dedis/student_19_proof-of-loc/blssig/latencyprotocol"
	uuid "github.com/satori/go.uuid"
	"go.dedis.ch/cothority/v3/messaging"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
	bbolt "go.etcd.io/bbolt"
	"sync"
	"time"
)

// This file contains all the code to run a BLSCoSi service. It is used to reply to
// client request for signing something using BLSCoSiService.
// This is an updated version of the CoSi Service (dedis/cothority/cosi/service), which only does simple signing

// ServiceName is the name to refer to BLSCoSiService
const ServiceName = "BLSCoSiService"

const blscosiSigProtocolName = "blscosiproto"

var serviceID onet.ServiceID

// BLSCoSiService is the service that handles collective signing operations
type BLSCoSiService struct {
	*onet.ServiceProcessor
	propagationFunction messaging.PropagationFunc
	propagatedSignature []byte
	Chain               *latencyprotocol.Chain
	Suite               *pairing.SuiteBn256
	Nodes               []*latencyprotocol.Node
}

func newBLSCoSiService(c *onet.Context) (onet.Service, error) {
	s := &BLSCoSiService{
		ServiceProcessor: onet.NewServiceProcessor(c),
		Chain:            &latencyprotocol.Chain{Blocks: make([]*latencyprotocol.Block, 0), BucketName: []byte("latencyprotocolBlocks")},
		Suite:            pairing.NewSuiteBn256(),
	}

	err := s.RegisterHandler(s.SignatureRequest)
	if err != nil {
		log.Error(err, "Couldn't register handler:")
		return nil, err
	}

	err = s.RegisterHandler(s.CreateBlock)
	if err != nil {
		log.Error(err, "Couldn't register handler:")
		return nil, err
	}

	err = s.RegisterHandler(s.CreateNode)
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

func init() {
	var err error
	serviceID, err = onet.RegisterNewService(ServiceName, newBLSCoSiService)
	log.ErrFatal(err)
	onet.GlobalProtocolRegister(blscosiSigProtocolName, blscosiprotocol.NewDefaultProtocol)
	network.RegisterMessages(&SignatureRequest{}, &SignatureResponse{})
	network.RegisterMessage(&PropagationFunction{})
	network.RegisterMessages(&CreateBlockRequest{}, &CreateBlockResponse{})
	network.RegisterMessages(&CreateNodeRequest{}, &CreateNodeResponse{})
}

// SignatureRequest treats external requests to this service.
func (s *BLSCoSiService) SignatureRequest(req *SignatureRequest) (*SignatureResponse, error) {
	sig, prop, err := s.sign(req.Roster, req.Message)
	if err != nil {
		return nil, err
	}
	return &SignatureResponse{sig, prop}, nil

}

func (s *BLSCoSiService) sign(Roster *onet.Roster, Message []byte) ([]byte, []byte, error) {

	if Roster.ID.IsNil() {
		Roster.ID = onet.RosterID(uuid.NewV4())
	}

	_, root := Roster.Search(s.ServerIdentity().ID)
	if root == nil {
		return nil, nil, errors.New("Couldn't find a serverIdentity in Roster")
	}

	tree := Roster.GenerateNaryTreeWithRoot(2, root)
	pi, err := s.CreateProtocol(blscosiSigProtocolName, tree)
	if err != nil {
		return nil, nil, errors.New("Couldn't make new protocol: " + err.Error())
	}

	//Set message and start signing
	protocolInstance := pi.(*blscosiprotocol.SimpleBLSCoSi)
	protocolInstance.Message = Message

	log.Lvl3("BLSCosi Service starting up root protocol")

	if err = pi.Start(); err != nil {
		return nil, nil, err
	}

	//Get signature
	sig := <-protocolInstance.FinalSignature

	// We propagate the signature to all nodes
	err = s.startPropagation(s.propagationFunction, Roster, &PropagationFunction{sig})
	if err != nil {
		log.Error(err, "Couldn't propagate signature:")
		return nil, nil, err
	}

	prop := s.propagatedSignature

	return sig, prop, nil

}

//CreateNode creates a new Block
func (s *BLSCoSiService) CreateNode(request *CreateNodeRequest) (*CreateNodeResponse, error) {
	id := request.ID

	newNode, shutdownChannel, err := latencyprotocol.NewNode(id, request.sendingAddress, s.Suite, request.nbLatenciesNeededForBlock)

	if err != nil {
		if shutdownChannel != nil {
			shutdownChannel <- true
		}
		return nil, err
	}

	s.Nodes = append(s.Nodes, newNode)

	nodeBytes, err := protobuf.Encode(newNode)

	if err != nil {
		if shutdownChannel != nil {
			shutdownChannel <- true
		}
		return nil, err
	}

	var wg sync.WaitGroup
	stopListeningForNewBlockChannel := make(chan bool, 1)

	wg.Add(1)
	go s.listenForNewBlocks(*newNode, stopListeningForNewBlockChannel, shutdownChannel, request.Roster, &wg)

	return &CreateNodeResponse{stopListeningForNewBlockChannel, nodeBytes}, nil

}

func (s *BLSCoSiService) listenForNewBlocks(node latencyprotocol.Node, stopListeningIncoming chan bool, stopListeningOutgoing chan bool,
	Roster *onet.Roster, wg *sync.WaitGroup) error {
	select {
	case <-stopListeningIncoming:
		stopListeningOutgoing <- true
		wg.Done()
		return nil
	case newBlock := <-node.BlockChannel:

		//do some work
		work(&node)

		blockBytes, err := protobuf.Encode(newBlock)
		if err != nil {
			break
		}

		sig, _, err := s.sign(Roster, blockBytes)
		if err != nil {
			break
		}

		h := sha256.New()
		h.Write(sig)

		//key is the hash of the block
		key := h.Sum([]byte{})

		//Add block to chain
		db, bucket := s.GetAdditionalBucket([]byte(s.Chain.BucketName))

		db.Update(func(tx *bbolt.Tx) error {
			tx.Bucket(bucket).Put(key, sig)
			return nil
		})

		//chain.Roster.Concat(block.id)
		s.Chain.Blocks = append(s.Chain.Blocks, &newBlock)
	}

	return nil
}

//CreateBlock adds a block to a chain
func (s *BLSCoSiService) CreateBlock(request *CreateBlockRequest) (*CreateBlockResponse, error) {

	node := latencyprotocol.Node{}
	err := protobuf.Decode(request.Node, &node)
	if err != nil {
		return nil, err
	}

	node.AddBlock(s.Chain)

	newBlock := <-node.BlockChannel

	//do some work
	work(&node)

	blockBytes, err := protobuf.Encode(newBlock)
	if err != nil {
		return nil, err
	}

	sig, _, err := s.sign(request.Roster, blockBytes)
	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write(sig)

	//key is the hash of the block
	key := h.Sum([]byte{})

	//Add block to chain
	db, bucket := s.GetAdditionalBucket([]byte(s.Chain.BucketName))

	db.Update(func(tx *bbolt.Tx) error {
		tx.Bucket(bucket).Put(key, sig)
		return nil
	})

	//chain.Roster.Concat(block.id)
	s.Chain.Blocks = append(s.Chain.Blocks, &newBlock)

	return &CreateBlockResponse{blockBytes}, nil
}

func work(node *latencyprotocol.Node) {

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
