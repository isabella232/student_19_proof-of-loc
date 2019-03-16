package proofofloc

import (
	"go.dedis.ch/onet/v3/network"
	"time"
)

//Nonce represents a random value to make a message unique
type Nonce int

// Block represents a block with unique identification and a list of pings
type Block struct {
	ID        *network.ServerIdentity
	Latencies map[*network.ServerIdentity]time.Duration
}

//Chain represents a list of blocks that have joined the system
type Chain struct {
	Blocks     []*Block
	BucketName []byte
}

//PingMsg represents a message sent to "ping" another validator
type PingMsg struct {
	ID           *network.ServerIdentity
	Nonce        Nonce
	IsReply      bool
	StartingTime time.Time
}
