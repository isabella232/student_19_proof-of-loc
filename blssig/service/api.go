package service

import (
	"errors"
	"github.com/dedis/student_19_proof-of-loc/blssig/proofofloc"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
	"math/rand"
	"time"
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

/*

ProposeNewBlock creates a new validator with given address and public key

Every time a new node joins the identity chain, i.e., creates a block, it uses the BLSCoSiService to have the block
signed by a majority, and then distributes it to other nodes. For now, nodes can join without doing any “work”,
but later we might add a “work” function, either computing a hash preimage like in Bitcoin or smth else.
In your design though, you can already take such an extension into account.

*/
func (c *Client) ProposeNewBlock(id *network.ServerIdentity, chain *proofofloc.Chain) (*proofofloc.Block, error) {

	latencies := make(map[*network.ServerIdentity]time.Duration)
	pending := make(map[*network.ServerIdentity]proofofloc.Nonce)

	//create new block
	newBlock := &proofofloc.Block{ID: id, Latencies: latencies, Nonces: pending, NbReplies: 0}

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
	nbPings := min(nbPingsNeeded, len(chain.Blocks))

	//for now just ping the first ones
	for i := 0; i < nbPings; i++ {
		//newBlock.Ping(c.blocks[i], c.suite) --for now just random delay
		randomDelay := time.Duration((rand.Intn(300-20) + 20)) * time.Millisecond

		newBlock.Latencies[chain.Blocks[i].ID] = randomDelay
		newBlock.NbReplies++
	}

	//wait till all reply
	for newBlock.NbReplies < nbPings {
		time.Sleep(1 * time.Millisecond)
	}

	return c.proposeBlock(newBlock, chain)

}

func (c *Client) proposeBlock(block *proofofloc.Block, chain *proofofloc.Chain) (*proofofloc.Block, error) {

	blockBytes, err := protobuf.Encode(block)
	if err != nil {
		return nil, err
	}

	chainBytes, err := protobuf.Encode(chain)
	if err != nil {
		return nil, err
	}

	c.SignatureRequest(chain.Roster, blockBytes)

	storageRequest := &StoreBlockRequest{
		Chain: chainBytes,
		Block: blockBytes,
	}

	if len(chain.Roster.List) == 0 {
		return nil, errors.New("Got an empty roster-list")
	}

	dst := chain.Roster.List[0]

	log.Lvl1("Sending message to", dst)
	reply := &StoreBlockResponse{}
	err = c.SendProtobuf(dst, storageRequest, reply)
	if err != nil {
		return nil, err
	}

	return block, nil
}
