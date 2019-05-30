/*
distance_test tests the functions implemented in the distance file to estimate the distance between two nodes
based on the measurements available in their chain
*/
package latencyprotocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3/log"
)

func TestApproximateDistanceAllInformation(t *testing.T) {

	N := 3
	x := 2

	chain, _ := initChain(N, x, accurate, 0, 0)

	_, exists := chain.Blocks[0].Latencies[string(chain.Blocks[1].ID.PublicKey)]

	log.Print(exists)

	d12, isValid12, err := chain.Blocks[0].ApproximateDistance(chain.Blocks[1], chain.Blocks[2], 10)

	require.Nil(t, err, "Error")
	require.Equal(t, d12, time.Duration(10*(1+2+1)))
	require.True(t, isValid12)

	d02, isValid02, err := chain.Blocks[1].ApproximateDistance(chain.Blocks[0], chain.Blocks[2], 10)

	require.Nil(t, err, "Error")
	require.Equal(t, d02, time.Duration(10*(2+1)))
	require.True(t, isValid02)

	d01, isValid01, err := chain.Blocks[2].ApproximateDistance(chain.Blocks[0], chain.Blocks[1], 10)

	require.Nil(t, err, "Error")
	require.Equal(t, d01, time.Duration(10*(1+1)))
	require.True(t, isValid01)

}

func TestApproximateDistanceInaccurateInformation(t *testing.T) {

	N := 6
	x := 4

	chain, _ := initChain(N, x, inaccurate, N, N)

	_, isValid, err := chain.Blocks[0].ApproximateDistance(chain.Blocks[1], chain.Blocks[2], 0)

	require.NotNil(t, err, "Inaccuracy error should have been reported")
	require.False(t, isValid)

}

func TestApproximateDistanceIncompleteInformation(t *testing.T) {

	/* Test Environment:

	N1---(d01 + d10/2)----N0----d02----N2

	N1-N2 unknown by any nodes -> pythagoras
	N0 - N2 only given by one node -> not trustworthy


	*/

	N := 3
	x := 1

	expectedD01 := time.Duration(10003 / 2)
	expectedD02 := time.Duration(((2 * 10000) + 1))
	expectedD12 := Pythagoras(expectedD01, expectedD02)

	chain, _ := initChain(N, x, inaccurate, N, N)

	d01, isValid01, err := chain.Blocks[2].ApproximateDistance(chain.Blocks[0], chain.Blocks[1], 10000)

	require.Nil(t, err, "Error")
	require.Equal(t, d01, expectedD01)
	require.True(t, isValid01)

	_, isValid02, err := chain.Blocks[1].ApproximateDistance(chain.Blocks[0], chain.Blocks[2], 10000)

	require.NotNil(t, err)
	require.False(t, isValid02)

	d12, isValid12, err := chain.Blocks[0].ApproximateDistance(chain.Blocks[1], chain.Blocks[2], 10000)

	require.Nil(t, err, "Error")
	require.Equal(t, d12, expectedD12)
	require.True(t, isValid12)

}

func TestApproximateDistanceMissingInformation(t *testing.T) {

	N := 5
	x := 1

	chain, _ := initChain(N, x, accurate, 0, 0)

	_, isValid, err := chain.Blocks[2].ApproximateDistance(chain.Blocks[3], chain.Blocks[4], 0)

	require.NotNil(t, err, "Should not have sufficient information to approximate distance")
	require.False(t, isValid)

}
