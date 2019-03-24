package service

import (
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// SignatureRequest is what the BLSCosi service is expected to receive from clients to sign stuff
type SignatureRequest struct {
	Message []byte
	Roster  *onet.Roster
}

// SignatureResponse is what the BLSCosi service will reply to clients who want stuff signed
type SignatureResponse struct {
	Signature  []byte
	Propagated []byte
}

// PropagationFunction sends the complete signature to all members of the Cothority
type PropagationFunction struct {
	Signature []byte
}

//CreateNodeRequest is what the BLSCosi service is expected to receive from clients to create a new block
type CreateNodeRequest struct {
	Roster *onet.Roster
	ID     *network.ServerIdentity
}

//CreateNodeResponse is what a BLSCosi service replies to clients trying to create blocks
type CreateNodeResponse struct {
	Node []byte
}

//StoreBlockRequest is what the BLSCosi service is expected to receive from clients to add Blocks to a chain
type StoreBlockRequest struct {
	Roster *onet.Roster
	Block  []byte
}

//StoreBlockResponse is what a BLSCosi service replies to clients trying to store blocks
type StoreBlockResponse struct {
	BlockAdded bool
}
