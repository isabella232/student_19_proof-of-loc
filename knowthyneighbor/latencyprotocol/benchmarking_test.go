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
	local.Check = onet.CheckNone
	// generate 3 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	_, el, _ := local.GenTree(4, false)
	defer local.CloseAll()

	chain := &Chain{make([]*Block, 0), []byte("testBucket")}

	newNode1, finish1, wg1, err := NewNode(el.List[0], el.List[1].Address, tSuite, 1)
	if err != nil {
		return nil, nil, err
	}

	chain.Blocks = append(chain.Blocks, &Block{newNode1.ID, make(map[string]ConfirmedLatency, 0)})

	newNode2, finish2, wg2, err := NewNode(el.List[2], el.List[3].Address, tSuite, 1)
	if err != nil {
		return nil, nil, err
	}

	newNode2.AddBlock(chain)

	block1 := <-newNode1.BlockChannel

	log.LLvl1("Channel 1 got its block")

	finish1 <- true
	wg1.Wait()

	if len(block1.Latencies) == 0 {
		return nil, nil, errors.New("Block 2 did not collect any latencies")
	}

	log.LLvl1("Adding blocks to chain")
	chain.Blocks = append(chain.Blocks, &block1)

	block2 := <-newNode2.BlockChannel

	log.LLvl1("Channel 2 got its block")

	finish2 <- true
	wg2.Wait()

	if len(block2.Latencies) == 0 {
		return nil, nil, errors.New("Block 2 did not collect any latencies")
	}

	chain.Blocks = append(chain.Blocks, &block2)

	log.LLvl1("Storing nodes")
	nodes := make([]*Node, 2)
	nodes[0] = newNode1
	nodes[1] = newNode2

	return nodes, chain, nil

}

func InterAddressPing(src *network.ServerIdentity, dst *network.ServerIdentity,
	srcAddress1 string, dstAddress1 string, srcAddress2 string, dstAddress2 string) (time.Duration, error) {

	var wg sync.WaitGroup

	msgChannel1, finishListeningChannel1, err := udp.InitListening(dstAddress1, &wg)
	if err != nil {
		return 0, err
	}
	msgChannel2, finishListeningChannel2, err := udp.InitListening(dstAddress2, &wg)
	if err != nil {
		return 0, err
	}

	msg := udp.PingMsg{}

	startTime1 := time.Now()
	finishSending1, sendMsgChan1 := udp.InitSending(srcAddress1, dstAddress1, &wg)
	sendMsgChan1 <- msg

	sentMsg := <-msgChannel1

	finishSending2, sendMsgChan2 := udp.InitSending(srcAddress2, dstAddress2, &wg)
	sendMsgChan2 <- sentMsg

	<-msgChannel2
	endTime1 := time.Now()

	finishListeningChannel1 <- true
	finishListeningChannel2 <- true
	finishSending1 <- true
	finishSending2 <- true

	wg.Wait()

	log.LLvl1("Both routines stopped")

	return endTime1.Sub(startTime1), nil

}

//Do this before running test on linux: sudo sysctl -w net.ipv4.ping_group_range="0   2147483647"
//Tool for slowing down latencies: https://bencane.com/2012/07/16/tc-adding-simulated-network-latency-to-your-linux-server/
func TestCompareLatenciesToPings(t *testing.T) {

	NbIterations := 2

	block1Latencies := make([]time.Duration, NbIterations)
	block2Latencies := make([]time.Duration, NbIterations)
	ping1Latencies := make([]time.Duration, NbIterations)
	ping2Latencies := make([]time.Duration, NbIterations)

	sumBlock1 := time.Duration(0)
	sumBlock2 := time.Duration(0)
	sumPing1 := time.Duration(0)
	sumPing2 := time.Duration(0)

	for i := 0; i < NbIterations; i++ {
		//time.Sleep(10 * time.Millisecond)
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
