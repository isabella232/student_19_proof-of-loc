package proofofloc

import (
	"go.dedis.ch/onet/v3/network"
	sigAlg "golang.org/x/crypto/ed25519"
	"time"
)

//Nonce represents a random value to make a message unique
type Nonce int

//IncompleteBlock represents a block in process of being constructed (latencies)
type IncompleteBlock struct {
	BlockSkeleton *Block
	PrivateKey    sigAlg.PrivateKey
	Nonces        map[*network.ServerIdentity][]byte
	NbReplies     *int
}

// Block represents a block with unique identification and a list of pings
type Block struct {
	ID        *network.ServerIdentity
	PublicKey sigAlg.PublicKey
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
	Nonce        []byte
	IsReply      bool
	StartingTime time.Time
}
