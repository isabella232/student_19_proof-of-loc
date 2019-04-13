package udp

import (
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	sigAlg "golang.org/x/crypto/ed25519"
	"sync"
	"testing"
	"time"
)

var tSuite = pairing.NewSuiteBn256()

func TestListeningInit(t *testing.T) {
	var wg sync.WaitGroup
	_, finish, err := InitListening("127.0.0.1:30001", &wg)
	finish <- true
	wg.Wait()
	require.NoError(t, err)
}

func TestSendingInit(t *testing.T) {
	var wg sync.WaitGroup
	finish, _ := InitSending("127.0.0.1:3001", "127.0.0.1:3002", &wg)
	finish <- true
	wg.Wait()
}

func TestMemoryLeaksCausedByLocal(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	local.Check = onet.CheckNone
	local.CloseAll()
	err := local.WaitDone(30 * time.Second)
	require.NoError(t, err, "Did not close all in time")
}

func TestSendOneMessage(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	local.Check = onet.CheckNone
	_, el, _ := local.GenTree(2, false)

	var wg sync.WaitGroup

	dstAddress := el.List[0].Address.NetworkAddress()
	srcAddress := el.List[1].Address.NetworkAddress()

	receptionChannel, finishListen, err := InitListening(dstAddress, &wg)

	require.NoError(t, err)

	pub, _, _ := sigAlg.GenerateKey(nil)

	msg := PingMsg{*el.List[0], *el.List[1], 10, pub, make([]byte, 0), make([]byte, 0)}

	finishSend, msgSending := InitSending(srcAddress, dstAddress, &wg)

	msgSending <- msg

	received := <-receptionChannel
	finishListen <- true
	finishSend <- true
	wg.Wait()

	require.NotNil(t, received)
	require.Equal(t, 10, received.SeqNb)
	local.CloseAll()

}

func TestSendTwoMessages(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	local.Check = onet.CheckNone

	_, el, _ := local.GenTree(2, false)

	var wg sync.WaitGroup

	dstAddress := el.List[1].Address.NetworkAddress()
	srcAddress := el.List[0].Address.NetworkAddress()

	receptionChannel, finishListen, err := InitListening(dstAddress, &wg)

	require.NoError(t, err)

	pub, _, _ := sigAlg.GenerateKey(nil)

	msg1 := PingMsg{*el.List[0], *el.List[1], 10, pub, make([]byte, 0), make([]byte, 0)}
	msg2 := PingMsg{*el.List[0], *el.List[1], 11, pub, make([]byte, 0), make([]byte, 0)}

	finishSend, msgSending := InitSending(srcAddress, dstAddress, &wg)

	msgSending <- msg1

	received1 := <-receptionChannel

	require.NotNil(t, received1)
	require.Equal(t, 10, received1.SeqNb)

	msgSending <- msg2

	received2 := <-receptionChannel

	finishListen <- true
	finishSend <- true
	wg.Wait()

	require.NotNil(t, received2)
	require.Equal(t, 11, received2.SeqNb)

	local.CloseAll()

}
