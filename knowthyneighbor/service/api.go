package service

import (
	"errors"

	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

const nbPingsNeeded = 5

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

ProposeNewNode creates a new validator with given address and public key

Every time a new node joins the identity chain, i.e., creates a block, it uses the BLSCoSiService to have the block
signed by a majority, and then distributes it to other nodes. For now, nodes can join without doing any “work”,
but later we might add a “work” function, either computing a hash preimage like in Bitcoin or smth else.

*/
func (c *Client) ProposeNewNode(id *network.ServerIdentity, roster *onet.Roster) error {

	if len(roster.List) == 0 {
		return errors.New("Got an empty roster-list")
	}

	dst := roster.List[0]

	newNodeRequest := &CreateNodeRequest{
		Roster: roster,
		ID:     id,
	}

	log.Lvl1("Sending node creation request message to", dst)
	createNodeReply := &CreateNodeResponse{}
	err := c.SendProtobuf(dst, newNodeRequest, createNodeReply)
	if err != nil {
		return err
	}

	return nil
}
