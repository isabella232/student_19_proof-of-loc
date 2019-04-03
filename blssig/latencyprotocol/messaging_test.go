package latencyprotocol

import (
	"errors"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"sync"
	"testing"
	"time"
)

const benchmarkDelta = 50 * time.Millisecond

func constructBlocks() ([]*Node, *Chain, error) {

	local := onet.NewTCPTest(tSuite)
	// generate 3 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, el, _ := local.GenTree(4, false)
	defer local.CloseAll()

	chain := &Chain{make([]*Block, 0), []byte("testBucket")}

	newNode1, finish1, err := NewNode(el.List[0], el.List[1].Address, tSuite, 1)
	if err != nil {
		return nil, nil, err
	}

	chain.Blocks = append(chain.Blocks, &Block{newNode1.ID, make(map[string]ConfirmedLatency, 0)})

	newNode2, finish2, err := NewNode(el.List[2], el.List[3].Address, tSuite, 1)
	if err != nil {
		return nil, nil, err
	}

	newNode2.AddBlock(chain)

	block1 := <-newNode1.BlockChannel

	log.LLvl1("Channel 1 got its block")

	finish1 <- true

	if len(block1.Latencies) == 0 {
		return nil, nil, errors.New("Block 2 did not collect any latencies")
	}

	log.LLvl1("Adding blocks to chain")
	chain.Blocks = append(chain.Blocks, &block1)

	block2 := <-newNode2.BlockChannel

	log.LLvl1("Channel 2 got its block")

	finish2 <- true

	if len(block2.Latencies) == 0 {
		return nil, nil, errors.New("Block 2 did not collect any latencies")
	}

	chain.Blocks = append(chain.Blocks, &block2)

	log.LLvl1(len(chain.Blocks))

	log.LLvl1("Storing nodes")
	nodes := make([]*Node, 2)
	nodes[0] = newNode1
	nodes[1] = newNode2

	return nodes, chain, nil

}

func InterAddressPing(srcAddress1 string, dstAddress1 string, srcAddress2 string, dstAddress2 string) (time.Duration, time.Duration, error) {

	var wg1 sync.WaitGroup
	var wg2 sync.WaitGroup
	finishListeningChannel1 := make(chan bool, 1)
	readyToListenChannel1 := make(chan bool, 1)
	finishListeningChannel2 := make(chan bool, 1)
	readyToListenChannel2 := make(chan bool, 1)

	msgChannel1 := InitListening(dstAddress1, finishListeningChannel1, readyToListenChannel1, &wg1)
	msgChannel2 := InitListening(dstAddress2, finishListeningChannel2, readyToListenChannel2, &wg2)

	<-readyToListenChannel1
	<-readyToListenChannel2

	msg1 := PingMsg{}
	msg2 := PingMsg{}

	startTime1 := time.Now()
	err := SendMessage(msg1, srcAddress1, dstAddress1)
	if err != nil {
		finishListeningChannel1 <- true
		finishListeningChannel2 <- true
		return 0, 0, err
	}

	<-msgChannel1
	endTime1 := time.Now()

	startTime2 := time.Now()
	err = SendMessage(msg2, srcAddress2, dstAddress2)
	if err != nil {
		finishListeningChannel1 <- true
		finishListeningChannel2 <- true
		return 0, 0, err
	}
	<-msgChannel2
	endTime2 := time.Now()

	finishListeningChannel1 <- true
	finishListeningChannel2 <- true

	wg1.Wait()
	wg2.Wait()

	return endTime1.Sub(startTime1), endTime2.Sub(startTime2), nil

}

//Do this before running test on linux: sudo sysctl -w net.ipv4.ping_group_range="0   2147483647"
//Tool ofr slowing down latencies: https://bencane.com/2012/07/16/tc-adding-simulated-network-latency-to-your-linux-server/
func TestCompareLatenciesToPings(t *testing.T) {

	log.LLvl1("Make chain")
	nodes, chain, err := constructBlocks()

	require.NoError(t, err)

	latency0, latency1, err := InterAddressPing(
		nodes[0].SendingAddress.NetworkAddress(),
		nodes[1].ID.ServerID.Address.NetworkAddress(),
		nodes[1].SendingAddress.NetworkAddress(),
		nodes[0].ID.ServerID.Address.NetworkAddress())
	require.NoError(t, err)

	expectedConfLat0, lat0here := chain.Blocks[1].Latencies[string(nodes[1].ID.PublicKey)]
	expectedConfLat1, lat1here := chain.Blocks[2].Latencies[string(nodes[0].ID.PublicKey)]

	expectedLat0 := expectedConfLat0.Latency
	expectedLat1 := expectedConfLat1.Latency

	require.True(t, lat0here, "Expected latency 1 not found")
	require.True(t, lat1here, "Expected latency 2 not found")

	log.LLvl1("Expected 1: " + expectedLat0.String())
	log.LLvl1("Actual 1: " + latency0.String())
	log.LLvl1("Difference: " + (expectedLat0 - latency0).String())

	log.LLvl1("Expected 2: " + expectedLat1.String())
	log.LLvl1("Actual 2: " + latency1.String())
	log.LLvl1("Difference: " + (expectedLat1 - latency1).String())

	require.True(t, (expectedLat0-latency0) < benchmarkDelta && (latency0-expectedLat0) < benchmarkDelta)
	require.True(t, (expectedLat1-latency1) < benchmarkDelta && (latency1-expectedLat1) < benchmarkDelta)
}
