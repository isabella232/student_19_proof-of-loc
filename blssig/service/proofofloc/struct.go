package proofofloc

import (
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
	"time"
)

//nonce represents a random value to make a message unique
type nonce int

// Block represents a block with unique identification and a list of pings
type Block struct {
	id        *network.ServerIdentity
	Latencies map[*network.ServerIdentity]time.Duration
	nonces    map[*network.ServerIdentity]nonce
	nbReplies int
}

//Chain represents a list of blocks that have joined the system
type Chain struct {
	suite  *pairing.SuiteBn256
	Roster *onet.Roster
	blocks []*Block
}

//PingMsg represents a message sent to "ping" another validator
type PingMsg struct {
	id           *network.ServerIdentity
	nonce        nonce
	isReply      bool
	startingTime time.Time
}
