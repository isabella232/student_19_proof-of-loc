package latencyprotocol

import (
	"github.com/stretchr/testify/require"
	//"go.dedis.ch/onet/v3/log"
	sigAlg "golang.org/x/crypto/ed25519"
	"testing"
)

func TestGetBestThreshBigDistance(t *testing.T) {
	set := NewBlacklistset()

	set.AddWithStrikes(sigAlg.PublicKey([]byte("A")), 1)
	set.AddWithStrikes(sigAlg.PublicKey([]byte("B")), 3)
	set.AddWithStrikes(sigAlg.PublicKey([]byte("C")), 24)
	set.AddWithStrikes(sigAlg.PublicKey([]byte("D")), 25)

	thresh, _ := set.GetBestThreshold()
	require.Equal(t, 24, thresh)

}

func TestGetBestThresholdSmallDistance(t *testing.T) {
	set := NewBlacklistset()

	set.AddWithStrikes(sigAlg.PublicKey([]byte("A")), 1)
	set.AddWithStrikes(sigAlg.PublicKey([]byte("B")), 2)
	set.AddWithStrikes(sigAlg.PublicKey([]byte("C")), 3)
	set.AddWithStrikes(sigAlg.PublicKey([]byte("D")), 5)
	set.AddWithStrikes(sigAlg.PublicKey([]byte("E")), 6)

	thresh, _ := set.GetBestThreshold()
	require.Equal(t, 5, thresh)

}

func TestGetBestThresholdAllSame(t *testing.T) {
	set := NewBlacklistset()

	set.AddWithStrikes(sigAlg.PublicKey([]byte("A")), 1)
	set.AddWithStrikes(sigAlg.PublicKey([]byte("B")), 1)
	set.AddWithStrikes(sigAlg.PublicKey([]byte("C")), 1)
	set.AddWithStrikes(sigAlg.PublicKey([]byte("D")), 1)
	set.AddWithStrikes(sigAlg.PublicKey([]byte("E")), 1)

	thresh, _ := set.GetBestThreshold()
	require.Equal(t, -1, thresh)

}
