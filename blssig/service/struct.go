package service

import (
	"github.com/dedis/student_19_proof-of-loc/blssig/proofofloc"
	"go.dedis.ch/onet/v3"
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

//StoreBlockRequest is what the BLSCosi service is expected to receive from clients to add Blocks to a chain
type StoreBlockRequest struct {
	Chain *proofofloc.Chain
	Block *proofofloc.Block
}

//StoreBlockResponse is what a BLSCosi service replies to clients trying to store blocks
type StoreBlockResponse struct {
	BlockAdded bool
}
