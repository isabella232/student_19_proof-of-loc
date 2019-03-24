package latencyprotocol

import (
	"errors"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
	sigAlg "golang.org/x/crypto/ed25519"
	"strconv"
	"time"
)

const nbLatencies = 5

//NewNode creates a new Node, initializes a new Block for the chain, and gets latencies for it
func NewNode(id *network.ServerIdentity, roster *onet.Roster, suite *pairing.SuiteBn256, chain *Chain) (*Node, error) {

	pubKey, privKey, err := sigAlg.GenerateKey(nil)
	if err != nil {
		return nil, err
	}

	nodeID := &NodeID{id, pubKey}

	latencies := make(map[string]ConfirmedLatency)

	//create new block
	newBlock := &Block{ID: nodeID, Latencies: latencies}

	receiverChannel := InitListening(id.Address.NetworkAddress())

	BlockChannel := make(chan Block, 1)

	newNode := &Node{
		ID:                      nodeID,
		PrivateKey:              privKey,
		LatenciesInConstruction: make(map[string]*LatencyConstructor),
		BlockSkeleton:           newBlock,
		NbLatenciesRefreshed:    0,
		IncomingMessageChannel:  receiverChannel,
		BlockChannel:            BlockChannel,
	}

	// send pings
	nbLatenciesNeeded := min(nbLatencies, len(roster.List))

	//this message loops forever handling incoming messages
	//its job is to put together latencies based on incoming messages and adding them to the block construction
	//When enough new latencies are collected, a new block is generated and sent in to be signed, and the process starts anew
	go handleIncomingMessages(newNode, nbLatenciesNeeded, chain)

	return newNode, nil

}

func (Node *Node) RefreshBlock(roster *onet.Roster, chain *Chain) {

	// send pings
	nbLatenciesNeeded := min(nbLatencies, len(roster.List))

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

func (A *Block) getLatency(B *Block) (time.Duration, bool) {
	latencyStruct, isPresent := A.Latencies[string(B.ID.PublicKey)]
	if !isPresent {
		return 0, false
	}
	return latencyStruct.Latency, true
}

/*ApproximateDistance is a function to approximate the distance between two given nodes, e.g.,
node A wants to approximate the distance between nodes B and C. Node A relies on the information
in the blockchain about distances to B, C, between B and C and its own estimations to B and C,
applies triangularization and computes an estimate of the distance. */
func (A *Block) ApproximateDistance(B *Block, C *Block, delta time.Duration) (time.Duration, error) {

	aToB, aToBKnown := A.getLatency(B)
	bToA, bToAKnown := B.getLatency(A)

	aToC, aToCKnown := A.getLatency(C)
	cToA, cToAKnown := C.getLatency(A)

	bToC, bToCKnown := B.getLatency(C)
	cToB, cToBKnown := C.getLatency(B)

	if cToBKnown && bToCKnown {
		if time.Duration(bToC-cToB) > delta || time.Duration(cToB-bToC) > delta {
			return time.Duration(0), errors.New("Distances contradictory: " + strconv.Itoa(int(time.Duration(bToC-cToB))))
		}
		return time.Duration((cToB + bToC) / 2), nil
	}

	if cToBKnown && !bToCKnown {
		return cToB, nil
	}

	if !cToBKnown && bToCKnown {
		return bToC, nil
	}

	if aToBKnown && bToAKnown {
		if time.Duration(aToB-bToA) > delta || time.Duration(bToA-aToB) > delta {
			return time.Duration(0), errors.New("Distances contradictory: " + strconv.Itoa(int(time.Duration(aToB-bToA))))
		}

		avgAB := (aToB + bToA) / 2

		if aToCKnown && cToAKnown {
			if time.Duration(aToC-cToA) > delta || time.Duration(cToA-aToC) > delta {
				return time.Duration(0), errors.New("Distances contradictory: " + strconv.Itoa(int(time.Duration(aToC-cToA))))
			}
			avgAC := (cToA + aToC) / 2

			return Pythagoras(avgAB, avgAC), nil

		}

		if aToCKnown && !cToAKnown {
			return Pythagoras(avgAB, aToC), nil

		}

		if !aToCKnown && cToAKnown {
			return Pythagoras(avgAB, cToA), nil

		}

	}

	if bToAKnown && !aToBKnown {
		if aToCKnown && cToAKnown {
			if time.Duration(aToC-cToA) > delta || time.Duration(cToA-aToC) > delta {
				return time.Duration(0), errors.New("Distances contradictory: " + strconv.Itoa(int(time.Duration(aToC-cToA))))
			}

			avgAC := (cToA + aToC) / 2

			return Pythagoras(bToA, avgAC), nil

		}

		if aToCKnown && !cToAKnown {
			return Pythagoras(bToA, aToC), nil

		}

		if !aToCKnown && cToAKnown {
			return Pythagoras(bToA, cToA), nil

		}

	}

	if !bToAKnown && aToBKnown {

		if aToCKnown && cToAKnown {
			if time.Duration(aToC-cToA) > delta || time.Duration(cToA-aToC) > delta {
				return time.Duration(0), errors.New("Distances contradictory: " + strconv.Itoa(int(time.Duration(aToC-cToA))))
			}

			avgAC := (cToA + aToC) / 2

			return Pythagoras(aToB, avgAC), nil

		}

		if aToCKnown && !cToAKnown {
			return Pythagoras(aToB, aToC), nil

		}

		if !aToCKnown && cToAKnown {
			return Pythagoras(aToB, cToA), nil

		}

	}

	return time.Duration(0), errors.New("Not enough information")

}

//Pythagoras estimates the distance between two points with known distances to a common third point b using the Pythagorean theorem
//Since the angle between the three points is between 0 and 180 degrees, the function assumes an average angle of 90 degreess
func Pythagoras(p1 time.Duration, p2 time.Duration) time.Duration {
	return ((p1 ^ 2) + (p2 ^ 2)) ^ (1 / 2)
}

func handleIncomingMessages(Node *Node, nbLatenciesForNewBlock int, chain *Chain) {

	for true {
		newMsg := <-Node.IncomingMessageChannel
		msgSeqNb := newMsg.SeqNb

		switch msgSeqNb {
		case 1:
			msgContent, messageOkay := Node.checkMessage1(&newMsg)
			if messageOkay {
				Node.sendMessage2(&newMsg, msgContent)
			}
		case 2:
			msgContent, messageOkay := Node.checkMessage2(&newMsg)
			if messageOkay {
				Node.sendMessage3(&newMsg, msgContent)
			}
		case 3:
			msgContent, messageOkay := Node.checkMessage3(&newMsg)
			if messageOkay {
				Node.sendMessage4(&newMsg, msgContent)
			}
		case 4:
			msgContent, messageOkay := Node.checkMessage4(&newMsg)
			if messageOkay {
				Node.sendMessage5(&newMsg, msgContent)
			}
		case 5:
			doubleSignedLatency, messageOkay := Node.checkMessage5(&newMsg)
			if messageOkay {
				Node.BlockSkeleton.Latencies[string(newMsg.PublicKey)] = *doubleSignedLatency

				//get rid of contructor
				Node.LatenciesInConstruction[string(newMsg.PublicKey)] = nil
				Node.NbLatenciesRefreshed++

				if Node.NbLatenciesRefreshed >= nbLatenciesForNewBlock {
					Node.BlockChannel <- *Node.BlockSkeleton
					Node.BlockSkeleton.Latencies = make(map[string]ConfirmedLatency)

				}
			}
		}

	}
}
