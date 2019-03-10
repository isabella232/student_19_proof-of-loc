package service

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/dedis/student_19_proof-of-loc/blssig/proofofloc"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

// Client is a structure to communicate with the BLSCoSi service
type Client struct {
	*onet.Client
}

// NewClient instantiates a new blscosi.Client
func NewClient() *Client {
	return &Client{Client: onet.NewClient(cothority.Suite, ServiceName)}
}

// SignatureRequest sends a CoSi sign request to the Cothority defined by the given
// Roster
func (c *Client) SignatureRequest(r *onet.Roster, msg []byte) (*SignatureResponse, error) {
	serviceReq := &SignatureRequest{
		Roster:  r,
		Message: msg,
	}
	if len(r.List) == 0 {
		return nil, errors.New("Got an empty roster-list")
	}
	dst := r.List[0]
	log.Lvl1("Sending message to", dst)
	reply := &SignatureResponse{}
	err := c.SendProtobuf(dst, serviceReq, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

//CreateNewChain allows a client to create a new chain
func (c *Client) CreateNewChain(suite *pairing.SuiteBn256) *Chain {
	return NewChain(suite)
}

/*

ProposeNewBlock creates a new validator with given address and public key

Every time a new node joins the identity chain, i.e., creates a block, it uses the BLSCoSiService to have the block
signed by a majority, and then distributes it to other nodes. For now, nodes can join without doing any “work”,
but later we might add a “work” function, either computing a hash preimage like in Bitcoin or smth else.
In your design though, you can already take such an extension into account.

*/
func (c *Client) ProposeNewBlock(id *network.ServerIdentity, chain *Chain) (*Block, error) {

	latencies := make(map[*network.ServerIdentity]time.Duration)
	pending := make(map[*network.ServerIdentity]nonce)

	//create new block
	newBlock := &Block{id, latencies, pending, 0}

	//get ping times from nodes

	//-> set up listening: disabled for now
	/*
		listener, err := network.NewTCPListener(id.Address, chain.suite)
		if err != nil {
			log.Error(err, "Couldn't create listener:")
			return nil, err
		}

		listener.Listen(newBlock.pingListen)
	*/

	// send pings
	nbPings := min(nbPingsNeeded, len(chain.blocks))

	//for now just ping the first ones
	for i := 0; i < nbPings; i++ {
		//newBlock.Ping(c.blocks[i], c.suite) --for now just random delay
		randomDelay := time.Duration((rand.Intn(300-20) + 20)) * time.Millisecond

		newBlock.Latencies[chain.blocks[i].id] = randomDelay
		newBlock.nbReplies++
	}

	//wait till all reply
	for newBlock.nbReplies < nbPings {
		time.Sleep(1 * time.Millisecond)
	}

	return c.ProposeBlock(newBlock, chain)

}

//ProposeBlock allows a client to propose an already formed block to a chain
func (c *Client) ProposeBlock(block *Block, chain *Chain) (*Block, error) {

	client := NewClient()

	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.BigEndian, block)
	if err != nil {
		return nil, err
	}
	fmt.Println(buf.Bytes())

	client.SignatureRequest(chain.Roster, buf.Bytes())

	//do some work
	block.work()

	//Add block to chain
	//chain.Roster.Concat(block.id)
	chain.blocks = append(chain.blocks, block)

	return block, nil
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
