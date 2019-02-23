package service

import (
	"testing"

	proofofloc "github.com/dedis/student_19_proof-of-loc/blssig"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

var testSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestService_Sign(t *testing.T) {
	local := onet.NewTCPTest(testSuite)
	// generate 5 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	hosts, roster, _ := local.GenTree(5, true)
	defer local.CloseAll()

	message := []byte("Message")
	services := local.GetServices(hosts, BLSCoSiServiceID)

	for _, s := range services {
		log.Lvl2("Sending request to", s)
		resp, err := s.(*SimpleBLSCoSiService).Sign(
			&proofofloc.Signed{
				Roster: roster,
				ToSign: message,
			},
		)
		require.Nil(t, err)
		require.NotNil(t, resp)
		//require.Equal(t, resp.Children, len(roster.List))
	}
}
