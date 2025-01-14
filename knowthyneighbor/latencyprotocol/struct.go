/*
struct contains necessary structures to execute the latency measurement protocol
*/

package latencyprotocol

import (
	"sync"
	"time"

	"github.com/dedis/student_19_proof-of-loc/knowthyneighbor/udp"
	"go.dedis.ch/onet/v3/network"
	sigAlg "golang.org/x/crypto/ed25519"
)

//Nonce represents a random value to make a message unique
type Nonce int

//NodeID represents an identifier for a node: its serverIdentity and Public Key
type NodeID struct {
	ServerID  *network.ServerIdentity
	PublicKey sigAlg.PublicKey
}

//LatencyWrapper wraps a latency because protobuf needs a struct
type LatencyWrapper struct {
	Latency time.Duration
}

//ConfirmedLatency is a struct that is stored in the block to represent latencies
type ConfirmedLatency struct {
	Latency            time.Duration
	SignedLatency      []byte
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
	SendingAddress          network.Address
	PrivateKey              sigAlg.PrivateKey
	LatenciesInConstruction map[string]*LatencyConstructor
	BlockSkeleton           *Block
	NbLatenciesRefreshed    int
	IncomingMessageChannel  chan udp.PingMsg
	BlockChannel            chan Block
}

//Chain represents a list of blocks that have joined the system
type Chain struct {
	Blocks     []*Block
	BucketName []byte
}

//Copy creates a deep copy of a chain
func (chain *Chain) Copy() *Chain {
	nbBlocks := len(chain.Blocks)
	blocksCopy := make([]*Block, nbBlocks)
	for i := 0; i < nbBlocks; i++ {
		latencies := make(map[string]ConfirmedLatency)
		for k, v := range chain.Blocks[i].Latencies {
			latencies[k] = v
		}
		blocksCopy[i] = &Block{
			ID: &NodeID{
				ServerID:  nil,
				PublicKey: sigAlg.PublicKey(numbersToNodes(i)),
			},
			Latencies: latencies,
		}

	}

	name := make([]byte, len(chain.BucketName))

	copy(name, chain.BucketName)

	return &Chain{blocksCopy, name}

}

//LatencyConstructor represents the values used during a latency calculation protocol
type LatencyConstructor struct {
	StartedLocally    bool
	CurrentMsgNb      int
	DstID             *NodeID
	Nonce             Nonce
	LocalTimestamps   []time.Time
	ForeignTimestamps []time.Time
	ClockSkews        []time.Duration
	Latency           time.Duration
	SignedLatency     []byte
	MsgChannel        *chan udp.PingMsg
	FinishedSending   *chan bool
	WaitGroup         *sync.WaitGroup
}
