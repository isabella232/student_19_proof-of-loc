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
func NewNode(id *network.ServerIdentity, sendingAddress network.Address, suite *pairing.SuiteBn256, nbLatencies int) (*Node, chan bool, error) {

	log.LLvl1("Creating new node")
	pubKey, privKey, err := sigAlg.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}

	nodeID := &NodeID{id, pubKey}

	latencies := make(map[string]ConfirmedLatency)

	//create new block
	newBlock := &Block{ID: nodeID, Latencies: latencies}

	var wg sync.WaitGroup
	udpReady := make(chan bool, 1)

	finish := make(chan bool, 1)
	finishListening := make(chan bool, 1)
	finishHandling := make(chan bool, 1)

	wg.Add(1)
	go passOnEndSignal(finish, finishHandling, finishListening, &wg)

	receiverChannel := udp.InitListening(id.Address.NetworkAddress(), finishListening, udpReady, &wg)

	<-udpReady

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

	return newNode, finish, nil

}

func passOnEndSignal(src chan bool, dst1 chan bool, dst2 chan bool, wg *sync.WaitGroup) {
	for {
		select {
		case <-src:
			log.LLvl1("Passing on end signal")
			dst1 <- true
			dst2 <- true
			wg.Done()
			return
		default:
		}
	}
}

//AddBlock lets a node add a new block to a chain
func (Node *Node) AddBlock(chain *Chain) {

	// send pings
	log.LLvl1("Adding Block")
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
				log.LLvl1("Incoming message 1")
				msgContent, messageOkay := Node.checkMessage1(&newMsg)
				if messageOkay {
					//Node.BlockSkeleton.ID = &NodeID{&newMsg.Src, newMsg.PublicKey}
					// TODO what if multiple latencies try to connect to node at same time -> Node needs list of blocks in construction
					err := Node.sendMessage2(&newMsg, msgContent)
					if err != nil {
						log.Warn(err.Error() + " - Could not send message: latency will not be recorded")
						Node.LatenciesInConstruction[string(newMsg.PublicKey)] = nil
					}
				}
			case 2:
				log.LLvl1("Incoming message 2")
				msgContent, messageOkay := Node.checkMessage2(&newMsg)
				if messageOkay {
					err := Node.sendMessage3(&newMsg, msgContent)
					if err != nil {
						log.Warn(err.Error() + " - Could not send message: latency will not be recorded")
						Node.LatenciesInConstruction[string(newMsg.PublicKey)] = nil
					}

				}
			case 3:
				log.LLvl1("Incoming message 3")
				msgContent, messageOkay := Node.checkMessage3(&newMsg)
				if messageOkay {
					err := Node.sendMessage4(&newMsg, msgContent)
					if err != nil {
						log.Warn(err.Error() + " - Could not send message: latency will not be recorded")
						Node.LatenciesInConstruction[string(newMsg.PublicKey)] = nil
					}
				}
			case 4:
				log.LLvl1("Incoming message 4")
				msgContent, messageOkay := Node.checkMessage4(&newMsg)
				if messageOkay {
					confirmedLatency, err := Node.sendMessage5(&newMsg, msgContent)
					encodedKey := string(newMsg.PublicKey)
					if err != nil {
						log.Warn(err.Error() + " - Could not send final message: latency will not be recorded")
					} else {
						log.LLvl1("Adding new latency to block")
						encodedKey := string(newMsg.PublicKey)
						Node.BlockSkeleton.Latencies[encodedKey] = *confirmedLatency //signature content, not whole message
						Node.NbLatenciesRefreshed++
					}

					Node.LatenciesInConstruction[encodedKey] = nil

					if Node.NbLatenciesRefreshed >= nbLatenciesForNewBlock && nbLatenciesForNewBlock > 0 {
						log.LLvl1("Sending up new block")
						Node.BlockChannel <- *Node.BlockSkeleton
						Node.BlockSkeleton.Latencies = make(map[string]ConfirmedLatency)

					}
				}

			case 5:
				log.LLvl1("Incoming message 5")
				doubleSignedLatency, messageOkay := Node.checkMessage5(&newMsg)
				if messageOkay {

					log.LLvl1("Adding new latency to block")
					encodedKey := string(newMsg.PublicKey)
					Node.BlockSkeleton.Latencies[encodedKey] = *doubleSignedLatency
					//get rid of contructor
					Node.LatenciesInConstruction[encodedKey] = nil
					Node.NbLatenciesRefreshed++

					if Node.NbLatenciesRefreshed >= nbLatenciesForNewBlock && nbLatenciesForNewBlock > 0 {
						log.LLvl1("Sending up new block")
						Node.BlockChannel <- *Node.BlockSkeleton
						Node.BlockSkeleton.Latencies = make(map[string]ConfirmedLatency)

					}
				}
			default:
				//log.LLvl1("Incorrect message id")
				//log.LLvl1(msgSeqNb)
				//return
			}
		}

	}

}
