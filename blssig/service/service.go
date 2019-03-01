// Package service implements a BLSCoSi service for which clients can connect to and then sign messages.
package service

import (
	"errors"
	"time"

	"github.com/dedis/student_19_proof-of-loc/blssig/protocol"
	uuid "github.com/satori/go.uuid"
	"go.dedis.ch/cothority/v3/messaging"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
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

// SignatureRequest is what the BLSCosi service is expected to receive from clients.
type SignatureRequest struct {
	Message []byte
	Roster  *onet.Roster
}

// SignatureResponse is what the BLSCosi service will reply to clients.
type SignatureResponse struct {
	Signature []byte
}

// PropagationFunction sends the complete signature to all members of the Cothority
type PropagationFunction struct {
	Signature []byte
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

	return &SignatureResponse{sig}, nil

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
