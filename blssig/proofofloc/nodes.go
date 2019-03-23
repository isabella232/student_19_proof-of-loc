package proofofloc

import (
	"errors"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
	sigAlg "golang.org/x/crypto/ed25519"
	"math/rand"
	"strconv"
	"time"
)

const nbLatencies = 5

//NewNode creates a new Node, initializes a new Block for the chain, and gets latencies for it
func NewNode(id *network.ServerIdentity, roster *onet.Roster, suite *pairing.SuiteBn256, chain *Chain) (*Node, error) {

	pubKey, privKey, err := sigAlg.GenerateKey(nil)

	nodeID := &NodeID{id, pubKey}

	latencies := make(map[*NodeID]Latency)

	//create new block
	newBlock := &Block{ID: nodeID, Latencies: latencies}

	newNode := &Node{
		ID:                      nodeID,
		PrivateKey:              privKey,
		LatenciesInConstruction: make([]LatencyConstructor, 0),
		BlockSkeleton:           newBlock,
	}

	//get ping times from nodes USE UDP ADD NONCE IN DATA -> 16byte + signed message in reply

	initConnection(id.Address, suite, newNode)

	// send pings
	nbLatenciesNeeded := min(nbLatencies, len(roster.List))

	//for now just ping the first ones
	for i := 0; i < nbLatenciesNeeded; i++ {
		newNode.LatenciesInConstruction[i] = LatencyConstructor{
			StartedLocally: true,
			DstID:          chain.Blocks[i].ID,
			Messages:       make([]PingMsg, 3),
			Nonces:         make([]byte, 2),
			Timestamps:     make([]time.Time, 2),
			ClockSkews:     make([]time.Duration, 2),
			latency:        0,
		}
	}

	nbReplies := 0
	//wait till all reply
	for len(newBlock.Latencies) < nbLatenciesNeeded {
		time.Sleep(1 * time.Millisecond)
	}

	return newNode, nil

}

func initConnection(address network.Address, suite *pairing.SuiteBn256, newNode *Node) {
	//get ping times from nodes USE UDP ADD NONCE IN DATA -> 16byte + signed message in reply

	listener, err := network.NewTCPListener(address, suite)
	if err != nil {
		log.Error(err, "Couldn't create listener:")
		return
	}

	listener.Listen(newNode.pingListen)

}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (A *Block) getLatency(B *Block) (time.Duration, bool) {
	latencyStruct, isPresent := A.Latencies[B.ID]
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

/*Ping allows a validator node to ping another node

The ping function is, for now, a random delay between 20 ms and 300 ms.

When node a pings node b, node a sends a message “ping” to node b (using onet) and node b replies with “pong” within a random delay time
*/
func (Node *Node) ping(dest *Node, suite *pairing.SuiteBn256) {

	conn, err := network.NewTCPConn(dest.ID.ServerID.Address, suite)

	if err != nil {
		log.Error(err, "Couldn't create new TCP connection:")
		return
	}

	nonceVal := rand.Int()
	nonceBytes, err := protobuf.Encode(nonceVal)
	if err != nil {
		return
	}

	//save nonce
	Node.Nonces[dest.ID] = nonceBytes

	_, err1 := conn.Send(PingMsg{ID: Node.BlockSkeleton.ID, Nonce: nonceBytes, IsReply: false, StartingTime: time.Now()})

	if err1 != nil {
		log.Error(err, "Couldn't send ping message:")
		return
	}

	conn.Close()

}

//pingListen listens for pings and pongs from other validators and handles them accordingly
func (Node *Node) pingListen(c network.Conn) {

	env, err := c.Receive()

	if err != nil {
		log.Error(err, "Couldn't send receive message from connection:")
		return
	}

	//Filter for the two types of messages we care about
	Msg, isPing := env.Msg.(PingMsg)

	// Case 1: someone pings us -> reply with pong and control values
	if isPing {
		if !Msg.IsReply {

			//CHECK TIMESTAMP FOR AGE of message

			signedNonce := sigAlg.Sign(Node.PrivateKey, Msg.Nonce)
			//sign return message, check time
			c.Send(PingMsg{ID: Node.BlockSkeleton.ID, Nonce: signedNonce, IsReply: true, StartingTime: Msg.StartingTime})
		} else {
			//Case 2: someone replies to our ping -> check return time
			if Msg.IsReply {

				nonceCorrect := sigAlg.Verify(Node.PublicKeys[Msg.ID], Msg.Nonce, Node.Nonces[Msg.ID])
				if nonceCorrect {

					latency := time.Since(Msg.StartingTime) //save start time locally
					Node.BlockSkeleton.Latencies[Msg.ID] = latency
					*Node.NbReplies++
				}

			}
		}

		c.Close()

	}
}
