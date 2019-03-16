package proofofloc

import (
	"errors"
	"time"
)

/*ApproximateDistance is a function to approximate the distance between two given nodes, e.g.,
node A wants to approximate the distance between nodes B and C. Node A relies on the information
in the blockchain about distances to B, C, between B and C and its own estimations to B and C,
applies triangularization and computes an estimate of the distance. Test it initially by assuming
all nodes are honest, each node adds in the blockchain x distances from itself to other x nodes,
where these x nodes are randomly chosen. You can assume for now that there’s a publicly known source
of randomness that nodes use. Check the results by varying the number x and the total number of nodes N.*/
func (A *Block) ApproximateDistance(B *Block, C *Block, delta time.Duration) (time.Duration, error) {

	aToB, aToBKnown := A.Latencies[B.ID]
	bToA, bToAKnown := B.Latencies[A.ID]

	aToC, aToCKnown := A.Latencies[C.ID]
	cToA, cToAKnown := C.Latencies[A.ID]

	bToC, bToCKnown := B.Latencies[C.ID]
	cToB, cToBKnown := C.Latencies[B.ID]

	if cToBKnown && bToCKnown {
		if time.Duration(bToC-cToB) > delta {
			return time.Duration(0), errors.New("B and C contradictory")
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
		if time.Duration(aToB-bToA) > delta {
			return time.Duration(0), errors.New("A and B contradictory")
		}

		avgAB := (aToB + bToA) / 2

		if aToCKnown && cToAKnown {
			if time.Duration(aToC-cToA) > delta {
				return time.Duration(0), errors.New("A and C contradictory")
			}
			avgAC := (cToA + aToC) / 2

			return pythagoras(avgAB, avgAC), nil

		}

		if aToCKnown && cToAKnown {
			if time.Duration(aToC-cToA) > delta {
				return time.Duration(0), errors.New("A and C contradictory")
			}

			avgAC := (cToA + aToC) / 2

			return pythagoras(avgAB, avgAC), nil

		}

		if aToCKnown && !cToAKnown {
			return pythagoras(avgAB, aToC), nil

		}

		if !aToCKnown && cToAKnown {
			return pythagoras(avgAB, cToA), nil

		}

	}

	if bToAKnown && !aToBKnown {
		if aToCKnown && cToAKnown {
			if time.Duration(aToC-cToA) > delta {
				return time.Duration(0), errors.New("A and C contradictory")
			}

			avgAC := (cToA + aToC) / 2

			return pythagoras(bToA, avgAC), nil

		}

		if aToCKnown && !cToAKnown {
			return pythagoras(bToA, aToC), nil

		}

		if !aToCKnown && cToAKnown {
			return pythagoras(bToA, cToA), nil

		}

	}

	if !bToAKnown && aToBKnown {

		if aToCKnown && cToAKnown {
			if time.Duration(aToC-cToA) > delta {
				return time.Duration(0), errors.New("A and C contradictory")
			}

			avgAC := (cToA + aToC) / 2

			return pythagoras(aToB, avgAC), nil

		}

		if aToCKnown && !cToAKnown {
			return pythagoras(aToB, aToC), nil

		}

		if !aToCKnown && cToAKnown {
			return pythagoras(aToB, cToA), nil

		}

	}

	return time.Duration(0), errors.New("Not enough information")

}

func pythagoras(p1 time.Duration, p2 time.Duration) time.Duration {
	return ((p1 ^ 2) + (p2 ^ 2)) ^ (1 / 2)
}
