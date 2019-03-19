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
	"time"
)

// This file contains all the code to run a BLSCoSi service. It is used to reply to
// client request for signing something using BLSCoSiService.
// This is an updated version of the CoSi Service (dedis/cothority/cosi/service), which only does simple signing

// ServiceName is the name to refer to BLSCoSiService
const ServiceName = "BLSCoSiService"

const protoName = "blscosiproto"

var serviceID onet.ServiceID

func init() {
	var err error
	serviceID, err = onet.RegisterNewService(ServiceName, newBLSCoSiService)
	log.ErrFatal(err)
	onet.GlobalProtocolRegister(protoName, protocol.NewDefaultProtocol)
	network.RegisterMessages(&SignatureRequest{}, &SignatureResponse{})
	network.RegisterMessage(&PropagationFunction{})
	network.RegisterMessages(&StoreBlockRequest{}, &StoreBlockResponse{})
	network.RegisterMessages(&CreateBlockRequest{}, &CreateBlockResponse{})
}

// BLSCoSiService is the service that handles collective signing operations
type BLSCoSiService struct {
	*onet.ServiceProcessor
	propagationFunction messaging.PropagationFunc
	propagatedSignature []byte
	Chain               *proofofloc.Chain
	Suite               *pairing.SuiteBn256
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

//CreateBlock creates a new Block
func (s *BLSCoSiService) CreateBlock(request *CreateBlockRequest) (*CreateBlockResponse, error) {
	roster := request.Roster
	id := request.ID

	newBlock, err := proofofloc.NewBlock(id, roster, s.Suite, s.Chain)

	blockBytes, err := protobuf.Encode(newBlock)
	if err != nil {
		return nil, err
	}

	return &CreateBlockResponse{blockBytes}, nil

}

//StoreBlock adds a block to a chain
func (s *BLSCoSiService) StoreBlock(request *StoreBlockRequest) (*StoreBlockResponse, error) {

	block := proofofloc.Block{}
	err := protobuf.Decode(request.Block, &block)
	if err != nil {
		return nil, err
	}
	//do some work
	work(&block)

	h := sha256.New()
	h.Write(request.Block)

	//key is the hash of the block
	key := h.Sum([]byte{})

	//Add block to chain
	db, bucket := s.GetAdditionalBucket([]byte(s.Chain.BucketName))

	db.Update(func(tx *bbolt.Tx) error {
		tx.Bucket(bucket).Put(key, request.Block)
		return nil
	})

	//chain.Roster.Concat(block.id)
	s.Chain.Blocks = append(s.Chain.Blocks, &block)

	return &StoreBlockResponse{true}, nil
}

func work(block *proofofloc.Block) {

}

func newBLSCoSiService(c *onet.Context) (onet.Service, error) {
	s := &BLSCoSiService{
		ServiceProcessor: onet.NewServiceProcessor(c),
		Chain:            &proofofloc.Chain{Blocks: make([]*proofofloc.Block, 0), BucketName: []byte("proofoflocBlocks")},
		Suite:            pairing.NewSuiteBn256(),
	}

	err := s.RegisterHandler(s.SignatureRequest)
	if err != nil {
		log.Error(err, "Couldn't register handler:")
		return nil, err
	}

	err = s.RegisterHandler(s.StoreBlock)
	if err != nil {
		log.Error(err, "Couldn't register handler:")
		return nil, err
	}

	err = s.RegisterHandler(s.CreateBlock)
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
