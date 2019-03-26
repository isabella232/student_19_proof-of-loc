package latencyprotocol

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	sigAlg "golang.org/x/crypto/ed25519"
	"sync"
	"testing"
)

const srcAddress = "127.0.0.1:2001"
const dstAddress = "127.0.0.1:2000"

func TestListeningInit(t *testing.T) {
	var wg sync.WaitGroup
	finish := make(chan bool, 1)
	ready := make(chan bool, 1)
	InitListening(dstAddress, finish, ready, &wg)
	readySig := <-ready
	finish <- true
	wg.Wait()
	require.True(t, readySig)
}

func TestSendOneMessage(t *testing.T) {
	local := onet.NewTCPTest(tSuite)

	_, el, _ := local.GenTree(2, false)

	defer local.CloseAll()

	var wg sync.WaitGroup
	finish := make(chan bool, 1)
	ready := make(chan bool, 1)

	receptionChannel := InitListening(dstAddress, finish, ready, &wg)

	pub, _, _ := sigAlg.GenerateKey(nil)

	msg := PingMsg{*el.List[0], *el.List[1], 10, pub, make([]byte, 0), make([]byte, 0)}

	err := SendMessage(msg, srcAddress, dstAddress)

	require.NoError(t, err)
	received := <-receptionChannel
	log.LLvl1("Got message")
	finish <- true
	wg.Wait()

	require.NotNil(t, received)
	require.Equal(t, 10, received.SeqNb)

}

func TestSendTwoMessages(t *testing.T) {
	local := onet.NewTCPTest(tSuite)

	_, el, _ := local.GenTree(2, false)

	defer local.CloseAll()

	var wg sync.WaitGroup
	finish := make(chan bool, 1)
	ready := make(chan bool, 1)

	receptionChannel := InitListening(dstAddress, finish, ready, &wg)

	pub, _, _ := sigAlg.GenerateKey(nil)

	msg1 := PingMsg{*el.List[0], *el.List[1], 10, pub, make([]byte, 0), make([]byte, 0)}
	msg2 := PingMsg{*el.List[0], *el.List[1], 11, pub, make([]byte, 0), make([]byte, 0)}

	err := SendMessage(msg1, srcAddress, dstAddress)

	require.NoError(t, err)
	received1 := <-receptionChannel
	log.LLvl1("Got message 1")

	require.NotNil(t, received1)
	require.Equal(t, 10, received1.SeqNb)

	err = SendMessage(msg2, srcAddress, dstAddress)
	require.NoError(t, err)
	received2 := <-receptionChannel
	log.LLvl1("Got message 2")

	finish <- true
	wg.Wait()

	require.NotNil(t, received2)
	require.Equal(t, 11, received2.SeqNb)

}
