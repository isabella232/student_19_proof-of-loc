package proofofloc

/*
This holds the messages used to communicate with the service over the network.
*/

import (
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// We need to register all messages so the network knows how to handle them.
func init() {
	network.RegisterMessages(
		Signed{}, SignedReply{},
	)
}

const (
	// ErrorParse indicates an error while parsing the protobuf-file.
	ErrorParse = iota + 4000
)

// Signed will return a signed message
type Signed struct {
	Roster *onet.Roster
	ToSign []byte
}

// SignedReply returns the signed message
type SignedReply struct {
	Signed []byte
}
