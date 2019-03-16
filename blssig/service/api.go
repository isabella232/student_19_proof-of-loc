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

// SignatureRequest sends a BLSCoSi sign request to the Cothority defined by the given
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

*/
func (c *Client) ProposeNewBlock(id *network.ServerIdentity, roster *onet.Roster) error {

	latencies := make(map[*network.ServerIdentity]time.Duration)
	//pending := make(map[*network.ServerIdentity]proofofloc.Nonce)

	nbReplies := 0

	//create new block
	newBlock := &proofofloc.Block{ID: id, Latencies: latencies}

	//get ping times from nodes USE UDP ADD NONCE IN DATA -> 16byte + signed message in reply

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
	nbPings := min(nbPingsNeeded, len(roster.List))

	//for now just ping the first ones
	for i := 0; i < nbPings; i++ {
		//newBlock.Ping(c.blocks[i], c.suite) --for now just random delay
		randomDelay := time.Duration((rand.Intn(300-20) + 20)) * time.Millisecond

		newBlock.Latencies[roster.List[i]] = randomDelay
		nbReplies++
	}

	//wait till all reply
	for nbReplies < nbPings {
		time.Sleep(1 * time.Millisecond)
	}

	return c.storeBlock(newBlock, roster)

}

func (c *Client) storeBlock(block *proofofloc.Block, roster *onet.Roster) error {

	blockBytes, err := protobuf.Encode(block)
	if err != nil {
		return err
	}

	c.SignatureRequest(roster, blockBytes)

	storageRequest := &StoreBlockRequest{
		Roster: roster,
		Block:  blockBytes,
	}

	if len(roster.List) == 0 {
		return errors.New("Got an empty roster-list")
	}

	dst := roster.List[0]

	log.Lvl1("Sending message to", dst)
	reply := &StoreBlockResponse{}
	err = c.SendProtobuf(dst, storageRequest, reply)
	if err != nil {
		return err
	}

	success := reply.BlockAdded

	if !success {
		return errors.New("Block not successfully added")
	}

	return nil
}
