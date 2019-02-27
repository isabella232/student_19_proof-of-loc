package proofofloc

import (
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3/network"
	"math/rand"
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

type nonce int

type pingcontrol struct {
	nonce      nonce
	returntime time.Time
}

type pingmsg struct {
	PublicKey kyber.Point
	Control   pingcontrol
}

type pongmsg struct {
	PublicKey kyber.Point
	Control   pingcontrol
}

var suite = pairing.NewSuiteBn256()

// Validator represents a validator with unique identification
type Validator struct {
	Address   string
	PublicKey kyber.Point
	Pings     map[kyber.Point]pingcontrol
	Listener  *network.TCPListener
}

// Block represents a block structure in a chain, containing validator identification
type Block struct {
	Validator     Validator
	NextBlock     *Block
	PreviousBlock *Block
}

// Chain represents a chain structure of blocks containing validator identifications
type Chain struct {
	FirstBlock *Block
	LastBlock  *Block
}

//Ping allows a validator node to ping another node
func (v Validator) Ping(b Validator) {
	delay := time.Duration((rand.Intn(300-20) + 20)) * time.Millisecond

	dst := network.NewTCPAddress(b.Address)

	conn, err := network.NewTCPConn(dst, suite)

	if err != nil {
		return
	}

	control := pingcontrol{nonce(rand.Int()), time.Now().Add(delay)}

	_, err1 := conn.Send(pingmsg{v.PublicKey, control})

	if err1 != nil {
		return
	}

	conn.Close()

	if err1 != nil {
		return
	}

	v.Pings[b.PublicKey] = control

}

//PingListen listens for pings and pongs from other validators and handles them accordingly
func (v Validator) PingListen(c network.Conn) {

	env, err := c.Receive()

	if err != nil {
		return
	}

	PingReceived, isPing := env.Msg.(pingmsg)
	PongReceived, isPong := env.Msg.(pongmsg)

	if isPing {
		c.Send(pongmsg{v.PublicKey, PingReceived.Control})
	} else {
		if isPong && v.Pings[PongReceived.PublicKey].nonce == PongReceived.Control.nonce {

			if PongReceived.Control.returntime.Before(time.Now()) {
				// what here?
			} else {
				//what here?
				delete(v.Pings, PongReceived.PublicKey)
			}
		}
	}

	c.Close()

}

func newValidator(Address string, PublicKey kyber.Point) (*Validator, error) {
	listener, err := network.NewTCPListener(network.NewTCPAddress(Address), suite)
	if err != nil {
		return nil, err
	}
	newVal := &Validator{Address, PublicKey, make(map[kyber.Point]pingcontrol), listener}

	listener.Listen(newVal.PingListen)

	return newVal, nil

}
