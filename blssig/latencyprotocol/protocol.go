package latencyprotocol

import (
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	sigAlg "golang.org/x/crypto/ed25519"
)

const nbLatencies = 5

//NewNode creates a new Node, initializes a new Block for the chain, and gets latencies for it
func NewNode(id *network.ServerIdentity, suite *pairing.SuiteBn256, nbLatencies int) (*Node, chan bool, error) {

	log.LLvl1("Creating new node")
	pubKey, privKey, err := sigAlg.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}

	nodeID := &NodeID{id, pubKey}

	latencies := make(map[string]ConfirmedLatency)

	//create new block
	newBlock := &Block{ID: nodeID, Latencies: latencies}

	udpReady := make(chan bool, 1)
	finish := make(chan bool, 1)

	finishListening := make(chan bool, 1)
	finishHandling := make(chan bool, 1)

	go passOnEndSignal(finish, finishHandling, finishListening)

	receiverChannel := InitListening(id.Address.NetworkAddress(), finishListening, udpReady)

	<-udpReady

	BlockChannel := make(chan Block, 1)

	newNode := &Node{
		ID:         nodeID,
		PrivateKey: privKey,
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
	go handleIncomingMessages(newNode, nbLatencies, finishHandling)

	return newNode, finish, nil

}

func passOnEndSignal(src chan bool, dst1 chan bool, dst2 chan bool) {
	for {
		select {
		case <-src:
			log.LLvl1("Passing on end signal")
			dst1 <- true
			dst2 <- true
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
	for i := 0; i < nbLatenciesNeeded; i++ {
		Node.sendMessage1(chain.Blocks[i].ID)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func handleIncomingMessages(Node *Node, nbLatenciesForNewBlock int, finish chan bool) {

	for {

		select {
		case <-finish:
			return
		default:
			newMsg := <-Node.IncomingMessageChannel
			log.LLvl1(newMsg)
			msgSeqNb := newMsg.SeqNb

			switch msgSeqNb {
			case 1:
				log.LLvl1("Incoming message 1")
				msgContent, messageOkay := Node.checkMessage1(&newMsg)
				if messageOkay {
					Node.sendMessage2(&newMsg, msgContent)
				}
			case 2:
				log.LLvl1("Incoming message 2")
				msgContent, messageOkay := Node.checkMessage2(&newMsg)
				if messageOkay {
					Node.sendMessage3(&newMsg, msgContent)
				}
			case 3:
				log.LLvl1("Incoming message 3")
				msgContent, messageOkay := Node.checkMessage3(&newMsg)
				if messageOkay {
					Node.sendMessage4(&newMsg, msgContent)
				}
			case 4:
				log.LLvl1("Incoming message 4")
				msgContent, messageOkay := Node.checkMessage4(&newMsg)
				if messageOkay {
					Node.sendMessage5(&newMsg, msgContent)
				}
			case 5:
				log.LLvl1("Incoming message 5")
				doubleSignedLatency, messageOkay := Node.checkMessage5(&newMsg)
				if messageOkay {
					Node.BlockSkeleton.Latencies[string(newMsg.PublicKey)] = *doubleSignedLatency

					//get rid of contructor
					Node.LatenciesInConstruction[string(newMsg.PublicKey)] = nil
					Node.NbLatenciesRefreshed++

					if Node.NbLatenciesRefreshed >= nbLatenciesForNewBlock && nbLatenciesForNewBlock > 0 {
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
