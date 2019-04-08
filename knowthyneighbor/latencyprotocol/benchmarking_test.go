package latencyprotocol

import (
	"errors"
	"github.com/dedis/student_19_proof-of-loc/knowthyneighbor/udp"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	//sigAlg "golang.org/x/crypto/ed25519"
	"math/rand"
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

func InterAddressPing(src *network.ServerIdentity, dst *network.ServerIdentity,
	srcAddress1 string, dstAddress1 string, srcAddress2 string, dstAddress2 string) (time.Duration, error) {

	var wg1 sync.WaitGroup
	var wg2 sync.WaitGroup
	finishListeningChannel1 := make(chan bool, 1)
	readyToListenChannel1 := make(chan bool, 1)
	finishListeningChannel2 := make(chan bool, 1)
	readyToListenChannel2 := make(chan bool, 1)

	msgChannel1 := udp.InitListening(dstAddress1, finishListeningChannel1, readyToListenChannel1, &wg1)
	msgChannel2 := udp.InitListening(dstAddress2, finishListeningChannel2, readyToListenChannel2, &wg2)

	<-readyToListenChannel1
	<-readyToListenChannel2

	/*pubKey, _, _ := sigAlg.GenerateKey(nil)

	msg := udp.PingMsg{
	Src:             *src,
	Dst:             *dst,
	SeqNb:           0,
	PublicKey:       pubKey,
	UnsignedContent: nil,
	SignedContent:   nil}*/

	msg := udp.PingMsg{}

	startTime1 := time.Now()
	err := udp.SendMessage(msg, srcAddress1, dstAddress1)
	if err != nil {
		finishListeningChannel1 <- true
		finishListeningChannel2 <- true
		return 0, err
	}

	sentMsg := <-msgChannel1

	err = udp.SendMessage(sentMsg, srcAddress2, dstAddress2)
	if err != nil {
		finishListeningChannel1 <- true
		finishListeningChannel2 <- true
		return 0, err
	}

	<-msgChannel2
	endTime1 := time.Now()

	finishListeningChannel1 <- true
	finishListeningChannel2 <- true

	wg1.Wait()
	wg2.Wait()

	return endTime1.Sub(startTime1), nil

}

//Do this before running test on linux: sudo sysctl -w net.ipv4.ping_group_range="0   2147483647"
//Tool for slowing down latencies: https://bencane.com/2012/07/16/tc-adding-simulated-network-latency-to-your-linux-server/
func TestCompareLatenciesToPings(t *testing.T) {

	NbIterations := 10

	block1Latencies := make([]time.Duration, 10)
	block2Latencies := make([]time.Duration, 10)
	ping1Latencies := make([]time.Duration, 10)
	ping2Latencies := make([]time.Duration, 10)

	sumBlock1 := time.Duration(0)
	sumBlock2 := time.Duration(0)
	sumPing1 := time.Duration(0)
	sumPing2 := time.Duration(0)

	for i := 0; i < NbIterations; i++ {
		time.Sleep(10 * time.Millisecond)
		rand.New(nil)
		log.LLvl1("Make chain")
		nodes, chain, err := constructBlocks()

		require.NoError(t, err)

		latency0, err := InterAddressPing(
			nodes[0].ID.ServerID,
			nodes[1].ID.ServerID,
			nodes[0].SendingAddress.NetworkAddress(),
			nodes[1].ID.ServerID.Address.NetworkAddress(),
			nodes[1].SendingAddress.NetworkAddress(),
			nodes[0].ID.ServerID.Address.NetworkAddress())
		require.NoError(t, err)

		latency1, err :=
			InterAddressPing(
				nodes[1].ID.ServerID,
				nodes[0].ID.ServerID,
				nodes[1].SendingAddress.NetworkAddress(),
				nodes[0].ID.ServerID.Address.NetworkAddress(),
				nodes[0].SendingAddress.NetworkAddress(),
				nodes[1].ID.ServerID.Address.NetworkAddress())
		require.NoError(t, err)

		expectedConfLat0, lat0here := chain.Blocks[1].Latencies[string(nodes[1].ID.PublicKey)]
		expectedConfLat1, lat1here := chain.Blocks[2].Latencies[string(nodes[0].ID.PublicKey)]

		require.True(t, lat0here)
		require.True(t, lat1here)

		expectedLat0 := expectedConfLat0.Latency
		expectedLat1 := expectedConfLat1.Latency

		block1Latencies[i] = expectedLat0
		block2Latencies[i] = expectedLat1
		ping1Latencies[i] = latency0
		ping2Latencies[i] = latency1

		sumBlock1 += expectedLat0
		sumBlock2 += expectedLat1
		sumPing1 += latency0
		sumPing2 += latency1

	}

	avgBlock1 := (sumBlock1 / time.Duration(NbIterations))
	avgBlock2 := (sumBlock2 / time.Duration(NbIterations))

	avgPing1 := (sumPing1 / time.Duration(NbIterations))
	avgPing2 := (sumPing2 / time.Duration(NbIterations))

	log.LLvl1("--------------------------------------------------")

	log.LLvl1("Average Block latency 1: 	" + avgBlock1.String())
	log.LLvl1("Average Ping latency 1: 	" + avgPing1.String())
	log.LLvl1("Difference: 				" + (avgBlock1 - avgPing1).String())

	log.LLvl1("--------------------------------------------------")

	log.LLvl1("Average Block latency 2: 	" + avgBlock2.String())
	log.LLvl1("Average Ping latency 2: 	" + avgPing2.String())
	log.LLvl1("Difference: 				" + (avgBlock2 - avgPing2).String())
}
