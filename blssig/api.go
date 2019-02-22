package proofofloc

/*
The api.go defines the methods that can be called from the outside. Most
of the methods will take a roster so that the service knows which nodes
it should work with.

This part of the service runs on the client or the app.
*/

import (
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

// ServiceName is used for registration on the onet.
const ServiceName = "SimpleBLSCoSiService"

// Client is a structure to communicate with the template
// service
type Client struct {
	*onet.Client
}

// NewClient instantiates a new template.Client
func NewClient() *Client {
	return &Client{Client: onet.NewClient(cothority.Suite, ServiceName)}
}

// Sign will return the given string signed by all the nodes
func (c *Client) Sign(r *onet.Roster, toSign []byte) ([]byte, error) {
	reply := &SignedReply{}
	dst := r.RandomServerIdentity()
	log.Lvl4("Sending message to", dst)
	err := c.SendProtobuf(dst, &Signed{r, toSign}, reply)
	if err != nil {
		return nil, err
	}
	return reply.Signed, nil
}
