package proofofloc

import (
	"errors"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	sigAlg "golang.org/x/crypto/ed25519"
	"math/rand"
	"strconv"
	"time"
)

const nbPingsNeeded = 5

func newBlock(id *network.ServerIdentity, roster *onet.Roster, suite network.Suite) (*Block, error) {
	latencies := make(map[*network.ServerIdentity]time.Duration)
	//pending := make(map[*network.ServerIdentity]proofofloc.Nonce)

	nbReplies := 0

	pubKey, privKey, err := sigAlg.GenerateKey(nil)

	//create new block
	newBlock := &Block{ID: id, PublicKey: pubKey, Latencies: latencies}

	//get ping times from nodes USE UDP ADD NONCE IN DATA -> 16byte + signed message in reply

	//-> set up listening: disabled for now

	listener, err := network.NewTCPListener(id.Address, suite)
	if err != nil {
		log.Error(err, "Couldn't create listener:")
		return nil, err
	}

	listener.Listen(newBlock.pingListen)

	// send pings
	nbPings := min(nbPingsNeeded, len(roster.List))

	//for now just ping the first ones
	for i := 0; i < nbPings; i++ {
		newBlock.ping(c.blocks[i], c.suite)
		newBlock.Latencies[roster.List[i]] = randomDelay
		nbReplies++
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
func (A *Block) ping(dest *Block, suite *pairing.SuiteBn256) {

	conn, err := network.NewTCPConn(dest.ID.Address, suite)

	if err != nil {
		log.Error(err, "Couldn't create new TCP connection:")
		return
	}

	nonce := Nonce(rand.Int())

	_, err1 := conn.Send(PingMsg{ID: A.ID, Nonce: nonce, IsReply: false, StartingTime: time.Now()})

	if err1 != nil {
		log.Error(err, "Couldn't send ping message:")
		return
	}

	conn.Close()

}

//pingListen listens for pings and pongs from other validators and handles them accordingly
func (A *Block) pingListen(c network.Conn, nonces map[*network.ServerIdentity]Nonce, nbReplies *int) {

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

			signedNonce := sigAlg.Sign(A.ID.GetPrivate(), Msg.Nonce)
			c.Send(PingMsg{ID: A.ID, Nonce: Msg.Nonce, IsReply: true, StartingTime: Msg.StartingTime})
		} else {
			//Case 2: someone replies to our ping -> check return time
			if Msg.IsReply && nonces[Msg.ID] == Msg.Nonce {

				latency := time.Since(Msg.StartingTime)
				A.Latencies[Msg.ID] = latency
				*nbReplies++

			}
		}

		c.Close()

	}
}
