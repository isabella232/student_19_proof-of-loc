// Package service implements a BLSCoSi service for which clients can connect to
// and then sign messages.
package service

import (
	"errors"

	"github.com/dedis/student_19_proof-of-loc/blssig/protocol"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	uuid "gopkg.in/satori/go.uuid.v1"
)

// This file contains all the code to run a BLSCoSi service. It is used to reply to
// client request for signing something using BLSCoSiService.
// This is an updated version of the CoSi Service (dedis/cothority/cosi/service), which only does simple signing

// ServiceName is the name to refer to BLSCoSiService
const ServiceName = "BLSCoSiService"

func init() {
	log.Lvl1("Service: init")
	onet.RegisterNewService(ServiceName, newBLSCoSiService)
	network.RegisterMessage(&SignatureRequest{})
	network.RegisterMessage(&SignatureResponse{})
}

// BLSCoSiService is the service that handles collective signing operations
type BLSCoSiService struct {
	*onet.ServiceProcessor
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

// SignatureRequest treats external request to this service.
func (blscosiservice *BLSCoSiService) SignatureRequest(req *SignatureRequest) (network.Message, error) {
	log.Lvl1("Service: SignatureRequest")
	if req.Roster.ID.IsNil() {
		req.Roster.ID = onet.RosterID(uuid.NewV4())
	}

	_, root := req.Roster.Search(blscosiservice.ServerIdentity().ID)
	if root == nil {
		return nil, errors.New("Couldn't find a serverIdentity in Roster")
	}

	tree := req.Roster.GenerateNaryTreeWithRoot(2, root)
	tni := blscosiservice.NewTreeNodeInstance(tree, tree.Root, protocol.Name)
	pi, err := protocol.NewDefaultProtocol(tni)
	if err != nil {
		return nil, errors.New("Couldn't make new protocol: " + err.Error())
	}
	blscosiservice.RegisterProtocolInstance(pi)

	//Set message and start signing
	protocolInstance := pi.(*protocol.SimpleBLSCoSi)
	protocolInstance.Message = req.Message
	protocolInstance.Start()

	log.Lvl1("BLSCosi Service starting up root protocol")
	go pi.Dispatch()
	go pi.Start()

	return &SignatureResponse{Signature: <-protocolInstance.FinalSignature}, nil
}

// NewProtocol is called on all nodes of a Tree (except the root, since it is
// the one starting the protocol) so it's the Service that will be called to
// generate the PI on all others node.
func (blscosiservice *BLSCoSiService) NewProtocol(tn *onet.TreeNodeInstance, conf *onet.GenericConfig) (onet.ProtocolInstance, error) {
	log.Lvl1("Service: NewProtocol")
	pi, err := protocol.NewDefaultProtocol(tn)
	return pi, err
}

func newBLSCoSiService(c *onet.Context) (onet.Service, error) {
	log.Lvl1("Service: newBLSCoSiService")
	s := &BLSCoSiService{
		ServiceProcessor: onet.NewServiceProcessor(c),
	}
	err := s.RegisterHandler(s.SignatureRequest)
	if err != nil {
		log.Error(err, "Couldn't register message:")
		return nil, err
	}
	return s, nil
}
