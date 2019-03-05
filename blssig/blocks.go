package proofofloc

import (
	"fmt"
	"github.com/dedis/student_19_proof-of-loc/blssig/service"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"math/rand"
	"time"
)

const nbPingsNeeded = 5

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
	suite    *pairing.SuiteBn256
	Roster   *onet.Roster
	blocks   []*Block
	nbBlocks int
}

//PingMsg represents a message sent to "ping" another validator
type PingMsg struct {
	id           *network.ServerIdentity
	nonce        nonce
	isReply      bool
	startingTime time.Time
}

//NewChain builds a new chain
func NewChain(suite *pairing.SuiteBn256) *Chain {
	return &Chain{suite, &onet.Roster{}, make([]*Block, 0), 0}
}

/*

NewBlock creates a new validator with given address and public key

Every time a new node joins the identity chain, i.e., creates a block, it uses the BLSCoSiService to have the block
signed by a majority, and then distributes it to other nodes. For now, nodes can join without doing any “work”,
but later we might add a “work” function, either computing a hash preimage like in Bitcoin or smth else.
In your design though, you can already take such an extension into account.

*/
func NewBlock(id *network.ServerIdentity, c *Chain) (*Block, error) {

	log.Lvl1("Setting up NewBlock")

	latencies := make(map[*network.ServerIdentity]time.Duration)
	pending := make(map[*network.ServerIdentity]nonce)

	//create new block
	newBlock := &Block{id, latencies, pending, 0}

	//get ping times from nodes

	//-> set up listening: disabled for now
	/*
		listener, err := network.NewTCPListener(id.Address, c.suite)
		if err != nil {
			log.Error(err, "Couldn't create listener:")
			return nil, err
		}

		listener.Listen(newBlock.pingListen)
	*/

	// send pings
	nbPings := min(nbPingsNeeded, c.nbBlocks)

	//for now just ping the first ones
	log.Lvl1("Pinging " + string(nbPings) + " nodes")
	for i := 0; i < nbPings; i++ {
		//newBlock.Ping(c.blocks[i], c.suite) --for now just random delay
		randomDelay := time.Duration((rand.Intn(300-20) + 20)) * time.Millisecond

		newBlock.Latencies[c.blocks[i].id] = randomDelay
		newBlock.nbReplies++
	}

	//wait till all reply
	for newBlock.nbReplies < nbPings {
		time.Sleep(1 * time.Millisecond)
	}

	log.Lvl1("Signing ")
	//sign new block
	client := service.NewClient()
	client.SignatureRequest(c.Roster, []byte(fmt.Sprintf("%v", newBlock)))

	log.Lvl1("Work")
	//do some work
	newBlock.work()

	log.Lvl1("Add to chain")
	//Add block to chain
	//c.Roster.Concat(id)
	c.blocks = append(c.blocks, newBlock)
	c.nbBlocks++

	return newBlock, nil

}

func (b *Block) work() {

}

//pingListen listens for pings and pongs from other validators and handles them accordingly
func (b *Block) pingListen(c network.Conn) {

	env, err := c.Receive()

	if err != nil {
		log.Error(err, "Couldn't send receive message from connection:")
		return
	}

	//Filter for the two types of messages we care about
	Msg, isPing := env.Msg.(PingMsg)

	// Case 1: someone pings ups -> reply with pong and control values
	if isPing {
		if !Msg.isReply {
			c.Send(PingMsg{b.id, Msg.nonce, true, Msg.startingTime})
		} else {
			//Case 2: someone replies to our ping -> check return time
			if Msg.isReply && b.nonces[Msg.id] == Msg.nonce {

				latency := time.Since(Msg.startingTime)
				b.Latencies[Msg.id] = latency
				b.nbReplies++

			}
		}

		c.Close()

	}
}

/*Ping allows a validator node to ping another node

The ping function is, for now, a random delay between 20 ms and 300 ms.

When node a pings node b, node a sends a message “ping” to node b (using onet) and node b replies with “pong” within a random delay time
*/
func (b *Block) Ping(dest *Block, suite *pairing.SuiteBn256) {

	//get random time delay between 20 and 300 ms - for now, just return this ----------------------------------------
	randomDelay := time.Duration((rand.Intn(300-20) + 20)) * time.Millisecond

	b.Latencies[dest.id] = randomDelay
	b.nbReplies++
	// -----------------------------------------------------------------------------------

	conn, err := network.NewTCPConn(dest.id.Address, suite)

	if err != nil {
		log.Error(err, "Couldn't create new TCP connection:")
		return
	}

	nonce := nonce(rand.Int())

	b.nonces[dest.id] = nonce

	_, err1 := conn.Send(PingMsg{b.id, nonce, false, time.Now()})

	if err1 != nil {
		log.Error(err, "Couldn't send ping message:")
		return
	}

	conn.Close()

}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
