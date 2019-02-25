package proofofloc

import (
	"container/list"
	"crypto/rsa"
	"go.dedis.ch/onet/v3/network"
	"math/rand"
	"net"
	"time"
)

/**
Implement a chain structure where blocks contain validator identification, such as IP, public key, and a list of pings to other validators

For now, have the validators choose the nodes they ping.
The ping function is, for now, a random delay between 20 ms and 300 ms.
When node a pings node b, node a sends a message “ping” to node b (using onet) and node b replies with “pong” within a random delay time
Every time a new node joins the identity chain, i.e., creates a block, it uses the service implemented above to have the block
signed by a majority, and then distributes it to other nodes. For now, nodes can join without doing any “work”,
but later we might add a “work” function, either computing a hash preimage like in Bitcoin or smth else.
In your design though, you can already take such an extension into account.

*/

// ValidatorID represents a validator's unique identification
type ValidatorID struct {
	IP        network.Address
	PublicKey kyber.Point
	Pings     list.List
}

// Block represents a block structure in a chain, containing validator identification
type Block struct {
	ValidatorID   ValidatorID
	NextBlock     *Block
	PreviousBlock *Block
}

// Chain represents a chain structure of blocks containing validator identifications
type Chain struct {
	FirstBlock *Block
	LastBlock  *Block
}

/*
 Using the ping function, a node can ping a chosen node.
*/
func (a ValidatorID) ping(b ValidatorID) {
	delay := rand.Int(20, 300)

	dest := network.ServerIdentity{
		b.PublicKey, b.IP,
	}

	a.Pings.PushBack(b)

}
