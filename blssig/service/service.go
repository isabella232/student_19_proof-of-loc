package service

/*
The service.go defines what to do for each API-call. This part of the service
runs on the node.
*/

import (
	"errors"
	"sync"

	proofofloc "github.com/dedis/student_19_proof-of-loc/blssig"
	"github.com/dedis/student_19_proof-of-loc/blssig/protocol"

	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

// BLSCoSiServiceID is the Service is used for tests
var BLSCoSiServiceID onet.ServiceID

func init() {
	var err error
	BLSCoSiServiceID, err = onet.RegisterNewService(proofofloc.ServiceName, newService)
	log.ErrFatal(err)
	network.RegisterMessage(&storage{})
}

// SimpleBLSCoSiService is our Service
type SimpleBLSCoSiService struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*onet.ServiceProcessor

	storage *storage
}

// storageID reflects the data we're storing - we could store more
// than one structure.
var storageID = []byte("main")

// storage is used to save our data.
type storage struct {
	Signed []byte
	sync.Mutex
}

// Sign returns the messaged signed by the instantiations of the protocol.
func (s *SimpleBLSCoSiService) Sign(req *proofofloc.Signed) (*proofofloc.SignedReply, error) {

	tree := req.Roster.GenerateNaryTreeWithRoot(2, s.ServerIdentity())
	if tree == nil {
		return nil, errors.New("couldn't create tree")
	}

	//Start protocol
	p, err := s.CreateProtocol(protocol.Name, tree)
	if err != nil {
		return nil, err

	}

	// Register the function generating the protocol instance
	var root *protocol.SimpleBLSCoSi

	root = p.(*protocol.SimpleBLSCoSi)
	root.Message = req.ToSign

	p.Start()

	resp := &proofofloc.SignedReply{
		Signed: <-p.(*protocol.SimpleBLSCoSi).FinalSignature,
	}

	return resp, nil
}

// NewProtocol is called on all nodes of a Tree (except the root, since it is
// the one starting the protocol) so it's the Service that will be called to
// generate the PI on all others node.
// If you use CreateProtocolOnet, this will not be called, as the Onet will
// instantiate the protocol on its own. If you need more control at the
// instantiation of the protocol, use CreateProtocolService, and you can
// give some extra-configuration to your protocol in here.
func (s *SimpleBLSCoSiService) NewProtocol(tn *onet.TreeNodeInstance, conf *onet.GenericConfig) (onet.ProtocolInstance, error) {
	return nil, nil
}

// saves all data.
func (s *SimpleBLSCoSiService) save() {
	s.storage.Lock()
	defer s.storage.Unlock()
	err := s.Save(storageID, s.storage)
	if err != nil {
		log.Error("Couldn't save data:", err)
	}
}

// Tries to load the configuration and updates the data in the service
// if it finds a valid config-file.
func (s *SimpleBLSCoSiService) tryLoad() error {
	s.storage = &storage{}
	msg, err := s.Load(storageID)
	if err != nil {
		return err
	}
	if msg == nil {
		return nil
	}
	var ok bool
	s.storage, ok = msg.(*storage)
	if !ok {
		return errors.New("Data of wrong type")
	}
	return nil
}

// newService receives the context that holds information about the node it's
// running on. Saving and loading can be done using the context. The data will
// be stored in memory for tests and simulations, and on disk for real deployments.
func newService(c *onet.Context) (onet.Service, error) {
	s := &SimpleBLSCoSiService{
		ServiceProcessor: onet.NewServiceProcessor(c),
	}
	if err := s.RegisterHandlers(s.Sign); err != nil {
		return nil, errors.New("Couldn't register messages")
	}
	if err := s.tryLoad(); err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}
