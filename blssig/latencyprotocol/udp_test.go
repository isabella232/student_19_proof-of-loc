package latencyprotocol

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	sigAlg "golang.org/x/crypto/ed25519"
	"testing"
)

const srcAddress = "127.0.0.1"
const dstAddress = ":2000"

func TestListeningInit(t *testing.T) {
	finish := make(chan bool)
	InitListening(dstAddress, finish)
	finish <- true
}

func TestSendMessage(t *testing.T) {
	local := onet.NewTCPTest(tSuite)

	_, el, _ := local.GenTree(12, false)

	defer local.CloseAll()

	finish := make(chan bool)

	receptionChannel := InitListening(dstAddress, finish)

	pub, _, _ := sigAlg.GenerateKey(nil)

	msg := PingMsg{*el.List[0], *el.List[1], 0, pub, make([]byte, 0), make([]byte, 0)}

	err := SendMessage(msg, srcAddress, dstAddress)

	require.NoError(t, err)
	received := <-receptionChannel
	log.LLvl1("Got message")

	require.NotNil(t, received)

	finish <- true

}
