package latencyprotocol

import (
	"go.dedis.ch/onet/v3/network"
	sigAlg "golang.org/x/crypto/ed25519"
	"time"
)

//Nonce represents a random value to make a message unique
type Nonce int

//NodeID represents an identifier for a node: its serverIdentity and Public Key
type NodeID struct {
	ServerID  *network.ServerIdentity
	PublicKey sigAlg.PublicKey
}

type ConfirmedLatency struct {
	Latency            time.Duration
	Timestamp          time.Time
	SignedConfirmation []byte
}

// Block represents a block with unique identification and a list of latencies of the following form: sigB[tsB, sigA[latABA]]
type Block struct {
	ID        *NodeID
	Latencies map[string]ConfirmedLatency
}

//Node represents a block in process of being constructed (latencies)
type Node struct {
	ID                      *NodeID
	PrivateKey              sigAlg.PrivateKey
	LatenciesInConstruction map[string]*LatencyConstructor
	BlockSkeleton           *Block
	NbLatenciesRefreshed    int
	IncomingMessageChannel  <-chan PingMsg
	BlockChannel            chan Block
}

//Chain represents a list of blocks that have joined the system
type Chain struct {
	Blocks     []*Block
	BucketName []byte
}

//LatencyConstructor represents the values used during a latency calculation protocol
type LatencyConstructor struct {
	StartedLocally bool
	CurrentMsgNb   int
	DstID          *NodeID
	Nonces         []Nonce
	Timestamps     []time.Time
	ClockSkews     []time.Duration
	Latency        time.Duration
}
