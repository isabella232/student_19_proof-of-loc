package proofofloc

import (
	"fmt"
	"github.com/dedis/student_19_proof-of-loc/blssig/service"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"math/rand"
	"time"
)

/**
Chain represents a chain structure where blocks contain validator identification, such as IP, public key, and a list of pings to other validators

For now, the validators choose the nodes they ping.
*/

//nonce represents a random value to make a message unique
type nonce int

//pingcontrol represents a unique message identifier and the time before which it should return
type pingcontrol struct {
	nonce      nonce
	returntime time.Time
}

//PingMsg represents a message sent to "ping" another validator
type PingMsg struct {
	PublicKey kyber.Point
	Control   pingcontrol
}

//PongMsg represents a message to reply to a ping
type PongMsg struct {
	PublicKey kyber.Point
	Control   pingcontrol
}

var suite = pairing.NewSuiteBn256()

// Block represents a block with unique identification and a list of pings
type Block struct {
	Address   network.Address
	PublicKey kyber.Point
	Pings     map[kyber.Point]pingcontrol
	Listener  *network.TCPListener
}

/*Ping allows a validator node to ping another node

The ping function is, for now, a random delay between 20 ms and 300 ms.

When node a pings node b, node a sends a message “ping” to node b (using onet) and node b replies with “pong” within a random delay time
*/
func (b Block) Ping(dest Block) {

	//get random time delay between 20 and 300 ms
	delay := time.Duration((rand.Intn(300-20) + 20)) * time.Millisecond

	conn, err := network.NewTCPConn(dest.Address, suite)

	if err != nil {
		log.Error(err, "Couldn't create new TCP connection:")
		return
	}

	//create control (Nonce, lastest return time) so we can uniquely identify a pong associated with a ping
	control := pingcontrol{nonce(rand.Int()), time.Now().Add(delay)}

	_, err1 := conn.Send(PingMsg{b.PublicKey, control})

	if err1 != nil {
		log.Error(err, "Couldn't send ping message:")
		return
	}

	conn.Close()

	// save the control to the validator
	b.Pings[dest.PublicKey] = control

}

//PingListen listens for pings and pongs from other validators and handles them accordingly
func (b Block) PingListen(c network.Conn) {

	env, err := c.Receive()

	if err != nil {
		log.Error(err, "Couldn't send receive message from connection:")
		return
	}

	//Filter for the two types of messages we care about
	PingReceived, isPing := env.Msg.(PingMsg)
	PongReceived, isPong := env.Msg.(PongMsg)

	// Case 1: someone pings ups -> reply with pong and control values
	if isPing {
		c.Send(PongMsg{b.PublicKey, PingReceived.Control})
	} else {
		//Case 2: someone replies to our ping -> check return time
		if isPong && b.Pings[PongReceived.PublicKey].nonce == PongReceived.Control.nonce {

			if PongReceived.Control.returntime.Before(time.Now()) {
				// what here?
			} else {
				//what here?
				delete(b.Pings, PongReceived.PublicKey)
			}
		}
	}

	c.Close()

}

/*

NewBlock creates a new validator with given address and public key

Every time a new node joins the identity chain, i.e., creates a block, it uses the BLSCoSiService to have the block
signed by a majority, and then distributes it to other nodes. For now, nodes can join without doing any “work”,
but later we might add a “work” function, either computing a hash preimage like in Bitcoin or smth else.
In your design though, you can already take such an extension into account.

*/
func NewBlock(id *network.ServerIdentity, r *onet.Roster) (*Block, error) {

	//make listener for incoming messages
	listener, err := network.NewTCPListener(id.Address, suite)
	if err != nil {
		log.Error(err, "Couldn't create listener:")
		return nil, err
	}

	newBlock := &Block{id.Address, id.Public, make(map[kyber.Point]pingcontrol), listener}

	client := service.NewClient()
	client.SignatureRequest(r, []byte(fmt.Sprintf("%v", newBlock)))

	newBlock.work()

	r.Concat(id)

	listener.Listen(newBlock.PingListen)

	return newBlock, nil

}

func (b *Block) work() {

}
