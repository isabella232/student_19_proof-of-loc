package proofofloc

import (
	"errors"
	"strconv"
	"time"
)

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
