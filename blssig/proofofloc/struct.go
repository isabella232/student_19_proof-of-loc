package proofofloc

import (
	"go.dedis.ch/onet/v3/network"
	sigAlg "golang.org/x/crypto/ed25519"
	"time"
)

//Nonce represents a random value to make a message unique
type Nonce []byte

//NodeID represents an identifier for a node: its serverIdentity and Public Key
type NodeID struct {
	ServerID  *network.ServerIdentity
	PublicKey sigAlg.PublicKey
}

type Latency struct {
	Latency            time.Duration
	Timestamp          time.Time
	SignedConfirmation []byte
}

// Block represents a block with unique identification and a list of latencies of the following form: sigB[tsB, sigA[latABA]]
type Block struct {
	ID        *NodeID
	Latencies map[*NodeID]Latency
}

//Node represents a block in process of being constructed (latencies)
type Node struct {
	ID                      *NodeID
	PrivateKey              sigAlg.PrivateKey
	LatenciesInConstruction []LatencyConstructor
	BlockSkeleton           *Block
}

//Chain represents a list of blocks that have joined the system
type Chain struct {
	Blocks     []*Block
	BucketName []byte
}

//PingMsg represents a message sent to another validator
type PingMsg struct {
	SrcIP string
	DstIP string
	SeqNb int

	content []byte
}

type PingMsg1 struct {
	PublicKey sigAlg.PublicKey
	SrcNonce  Nonce
	Timestamp time.Time
	Signed    []byte
}

type PingMsg2 struct {
	PublicKey sigAlg.PublicKey
	SrcNonce  Nonce
	DstNonce  Nonce
	Timestamp time.Time
	Signed    []byte
}

type PingMsg3 struct {
	PublicKey     sigAlg.PublicKey
	SrcNonce      Nonce
	DstNonce      Nonce
	Timestamp     time.Time
	Latency       time.Duration
	SignedLatency []byte
	Signed        []byte
}

type PingMsg4 struct {
	PublicKey           sigAlg.PublicKey
	SrcNonce            Nonce
	DstNonce            Nonce
	Timestamp           time.Time
	Latency             time.Duration
	SignedLatency       []byte
	DoubleSignedLatency []byte
	Signed              []byte
}

type PingMsg5 struct {
	PublicKey           sigAlg.PublicKey
	DstNonce            Nonce
	Timestamp           time.Time
	DoubleSignedLatency []byte
	Signed              []byte
}

//LatencyConstructor represents the values used during a latency calculation protocol
type LatencyConstructor struct {
	StartedLocally bool
	CurrentMsgNb   int
	DstID          *NodeID
	Messages       []PingMsg
	Nonces         []byte
	Timestamps     []time.Time
	ClockSkews     []time.Duration
	latency        time.Duration
}
