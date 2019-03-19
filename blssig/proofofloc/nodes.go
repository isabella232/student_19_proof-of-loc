package proofofloc

import (
	"bytes"
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

const nbPingsNeeded = 5

//NewBlock creates a new Block, gets latencies for it and returns
func NewBlock(id *network.ServerIdentity, roster *onet.Roster, suite *pairing.SuiteBn256, chain *Chain) (*Block, error) {

	latencies := make(map[*network.ServerIdentity]time.Duration)
	pending := make(map[*network.ServerIdentity][]byte)
	publicKeys := make(map[*network.ServerIdentity]sigAlg.PublicKey)

	for i := 0; i < len(chain.Blocks); i++ {
		publicKeys[chain.Blocks[i].ID] = chain.Blocks[i].PublicKey
	}

	nbReplies := 0

	pubKey, privKey, err := sigAlg.GenerateKey(nil)

	//create new block
	newBlock := &Block{ID: id, PublicKey: pubKey, Latencies: latencies}
	newBlockBuilder := &IncompleteBlock{
		BlockSkeleton: newBlock,
		PrivateKey:    privKey,
		Nonces:        pending,
		PublicKeys:    publicKeys,
		NbReplies:     &nbReplies,
	}

	//get ping times from nodes USE UDP ADD NONCE IN DATA -> 16byte + signed message in reply

	listener, err := network.NewTCPListener(id.Address, suite)
	if err != nil {
		log.Error(err, "Couldn't create listener:")
		return nil, err
	}

	listener.Listen(newBlockBuilder.pingListen)

	// send pings
	nbPings := min(nbPingsNeeded, len(roster.List))

	//for now just ping the first ones
	for i := 0; i < nbPings; i++ {
		newBlockBuilder.ping(chain.Blocks[i], suite)
	}

	//wait till all reply
	for nbReplies < nbPings {
		time.Sleep(1 * time.Millisecond)
	}

	return newBlock, nil

}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

/*ApproximateDistance is a function to approximate the distance between two given nodes, e.g.,
node A wants to approximate the distance between nodes B and C. Node A relies on the information
in the blockchain about distances to B, C, between B and C and its own estimations to B and C,
applies triangularization and computes an estimate of the distance. */
func (A *Block) ApproximateDistance(B *Block, C *Block, delta time.Duration) (time.Duration, error) {

	aToB, aToBKnown := A.Latencies[B.ID]
	bToA, bToAKnown := B.Latencies[A.ID]

	aToC, aToCKnown := A.Latencies[C.ID]
	cToA, cToAKnown := C.Latencies[A.ID]

	bToC, bToCKnown := B.Latencies[C.ID]
	cToB, cToBKnown := C.Latencies[B.ID]

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
func (BlockBuilder *IncompleteBlock) ping(dest *Block, suite *pairing.SuiteBn256) {

	conn, err := network.NewTCPConn(dest.ID.Address, suite)

	if err != nil {
		log.Error(err, "Couldn't create new TCP connection:")
		return
	}

	nonce := Nonce(rand.Int())
	nonceBytes, err := protobuf.Encode(nonce)
	if err != nil {
		return
	}

	//save nonce
	BlockBuilder.Nonces[dest.ID] = nonceBytes

	_, err1 := conn.Send(PingMsg{ID: BlockBuilder.BlockSkeleton.ID, Nonce: nonceBytes, IsReply: false, StartingTime: time.Now()})

	if err1 != nil {
		log.Error(err, "Couldn't send ping message:")
		return
	}

	conn.Close()

}

//pingListen listens for pings and pongs from other validators and handles them accordingly
func (BlockBuilder *IncompleteBlock) pingListen(c network.Conn) {

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

			signedNonce := sigAlg.Sign(BlockBuilder.PrivateKey, Msg.Nonce)
			c.Send(PingMsg{ID: BlockBuilder.BlockSkeleton.ID, Nonce: signedNonce, IsReply: true, StartingTime: Msg.StartingTime})
		} else {
			//Case 2: someone replies to our ping -> check return time
			if Msg.IsReply {
				nonceCorrect := sigAlg.Verify(BlockBuilder.PublicKeys[Msg.ID], Msg.Nonce, BlockBuilder.Nonces[Msg.ID])
				if nonceCorrect {

					latency := time.Since(Msg.StartingTime)
					BlockBuilder.BlockSkeleton.Latencies[Msg.ID] = latency
					*BlockBuilder.NbReplies++
				}

			}
		}

		c.Close()

	}
}
