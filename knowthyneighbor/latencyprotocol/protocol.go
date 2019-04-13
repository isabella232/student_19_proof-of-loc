package latencyprotocol

import (
	"github.com/dedis/student_19_proof-of-loc/knowthyneighbor/udp"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	sigAlg "golang.org/x/crypto/ed25519"
	"sync"
)

const nbLatencies = 5

//NewNode creates a new Node, initializes a new Block for the chain, and gets latencies for it
func NewNode(id *network.ServerIdentity, sendingAddress network.Address,
	suite *pairing.SuiteBn256, nbLatencies int) (*Node, chan bool, *sync.WaitGroup, error) {

	//this is what takes time
	pubKey, privKey, err := sigAlg.GenerateKey(nil)
	if err != nil {
		return nil, nil, nil, err
	}

	nodeID := &NodeID{id, pubKey}

	latencies := make(map[string]ConfirmedLatency)

	//create new block
	newBlock := &Block{ID: nodeID, Latencies: latencies}

	var wg sync.WaitGroup

	finish := make(chan bool, 1)
	finishHandling := make(chan bool, 1)

	receiverChannel, finishListening, err := udp.InitListening(id.Address.NetworkAddress(), &wg)

	if err != nil {
		return nil, nil, nil, err
	}

	wg.Add(1)
	go passOnEndSignal(finish, finishHandling, finishListening, &wg)

	BlockChannel := make(chan Block, 1)

	newNode := &Node{
		ID:             nodeID,
		SendingAddress: sendingAddress,
		PrivateKey:     privKey,
		//note: this takes a publicKey converted to a string as key
		LatenciesInConstruction: make(map[string]*LatencyConstructor),
		BlockSkeleton:           newBlock,
		NbLatenciesRefreshed:    0,
		IncomingMessageChannel:  receiverChannel,
		BlockChannel:            BlockChannel,
	}

	//this message loops forever handling incoming messages
	//its job is to put together latencies based on incoming messages and adding them to the block construction
	//When enough new latencies are collected, a new block is generated and sent in to be signed, and the process starts anew
	wg.Add(1)
	go handleIncomingMessages(newNode, nbLatencies, finishHandling, &wg)

	return newNode, finish, &wg, nil

}

func passOnEndSignal(src chan bool, dst1 chan bool, dst2 chan bool, wg *sync.WaitGroup) {
	select {
	case <-src:
		dst1 <- true
		dst2 <- true
		wg.Done()
		return
	}
}

//AddBlock lets a node add a new block to a chain
func (Node *Node) AddBlock(chain *Chain) {

	// send pings
	nbLatenciesNeeded := min(nbLatencies, len(chain.Blocks))

	//for now just ping the first ones
	for i := 0; i < nbLatenciesNeeded && i < len(chain.Blocks); i++ {
		Node.sendMessage1(chain.Blocks[i].ID)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func handleIncomingMessages(Node *Node, nbLatenciesForNewBlock int, finish chan bool, wg *sync.WaitGroup) {

	for {

		select {
		case <-finish:
			wg.Done()
			return
		case newMsg := <-Node.IncomingMessageChannel:
			msgSeqNb := newMsg.SeqNb

			switch msgSeqNb {
			case 1:
				msgContent, messageOkay := Node.checkMessage1(&newMsg)
				if messageOkay {
					err := Node.sendMessage2(&newMsg, msgContent)
					if err != nil {
						log.Warn(err.Error() + " - Could not send message: latency will not be recorded")
						Node.LatenciesInConstruction[string(newMsg.PublicKey)] = nil
					}
				}
			case 2:
				msgContent, messageOkay := Node.checkMessage2(&newMsg)
				if messageOkay {
					err := Node.sendMessage3(&newMsg, msgContent)
					if err != nil {
						log.Warn(err.Error() + " - Could not send message: latency will not be recorded")
						Node.LatenciesInConstruction[string(newMsg.PublicKey)] = nil
					}

				}
			case 3:
				msgContent, messageOkay := Node.checkMessage3(&newMsg)
				if messageOkay {
					err := Node.sendMessage4(&newMsg, msgContent)
					if err != nil {
						log.Warn(err.Error() + " - Could not send message: latency will not be recorded")
						Node.LatenciesInConstruction[string(newMsg.PublicKey)] = nil
					}
				}
			case 4:
				msgContent, messageOkay := Node.checkMessage4(&newMsg)
				if messageOkay {
					confirmedLatency, err := Node.sendMessage5(&newMsg, msgContent)
					encodedKey := string(newMsg.PublicKey)
					if err != nil {
						log.Warn(err.Error() + " - Could not send final message: latency will not be recorded")
					} else {
						encodedKey := string(newMsg.PublicKey)
						Node.BlockSkeleton.Latencies[encodedKey] = *confirmedLatency //signature content, not whole message
						Node.NbLatenciesRefreshed++
					}

					Node.LatenciesInConstruction[encodedKey] = nil

					if Node.NbLatenciesRefreshed >= nbLatenciesForNewBlock && nbLatenciesForNewBlock > 0 {
						Node.BlockChannel <- *Node.BlockSkeleton
						Node.BlockSkeleton.Latencies = make(map[string]ConfirmedLatency)

					}
				}

			case 5:
				doubleSignedLatency, messageOkay := Node.checkMessage5(&newMsg)
				if messageOkay {
					encodedKey := string(newMsg.PublicKey)
					Node.BlockSkeleton.Latencies[encodedKey] = *doubleSignedLatency
					//get rid of contructor
					Node.LatenciesInConstruction[encodedKey] = nil
					Node.NbLatenciesRefreshed++

					if Node.NbLatenciesRefreshed >= nbLatenciesForNewBlock && nbLatenciesForNewBlock > 0 {
						Node.BlockChannel <- *Node.BlockSkeleton
						Node.BlockSkeleton.Latencies = make(map[string]ConfirmedLatency)

					}
				}
			}
		}

	}

}
