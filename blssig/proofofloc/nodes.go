package proofofloc

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

	newNode := &Node{
		ID:                      nodeID,
		PrivateKey:              privKey,
		LatenciesInConstruction: make(map[string]*LatencyConstructor),
		BlockSkeleton:           newBlock,
		ReceptionChannel:        receiverChannel,
	}

	//get ping times from nodes USE UDP ADD NONCE IN DATA -> 16byte + signed message in reply

	// send pings
	nbLatenciesNeeded := min(nbLatencies, len(roster.List))

	//for now just ping the first ones
	for i := 0; i < nbLatenciesNeeded; i++ {
		newNode.sendMessage1(chain.Blocks[i].ID)
	}
	//wait till all reply
	for len(newBlock.Latencies) < nbLatenciesNeeded {
		time.Sleep(1 * time.Millisecond)
	}

	return newNode, nil

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
