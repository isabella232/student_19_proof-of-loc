package latencyprotocol

import (
	"go.dedis.ch/onet/v3/log"
	"testing"
)

func TestBlacklist1Liar1Victim(t *testing.T) {
	N := 4

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "A", "D", 25)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs, false))

}

func TestBlacklist2Liars3Victims(t *testing.T) {
	N := 8

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "A", "C", 25)
	setLiarAndVictim(chain, "B", "C", 25)
	setLiarAndVictim(chain, "A", "D", 25)
	setLiarAndVictim(chain, "B", "D", 25)
	setLiarAndVictim(chain, "A", "E", 25)
	setLiarAndVictim(chain, "B", "E", 25)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs, false))

}

func TestAverageBlacklist2Liars3Victims(t *testing.T) {
	N := 8

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "A", "C", 25)
	setLiarAndVictim(chain, "B", "C", 25)
	setLiarAndVictim(chain, "A", "D", 25)
	setLiarAndVictim(chain, "B", "D", 25)
	setLiarAndVictim(chain, "A", "E", 25)
	setLiarAndVictim(chain, "B", "E", 25)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs, true))

}

func TestBlacklist13Liars3Victims(t *testing.T) {
	N := 9

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "A", "D", 25)
	setLiarAndVictim(chain, "A", "E", 25)
	setLiarAndVictim(chain, "A", "F", 25)
	setLiarAndVictim(chain, "B", "G", 25)
	setLiarAndVictim(chain, "B", "H", 25)
	setLiarAndVictim(chain, "B", "I", 25)
	setLiarAndVictim(chain, "C", "D", 25)
	setLiarAndVictim(chain, "C", "E", 25)
	setLiarAndVictim(chain, "C", "G", 25)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs, false))

}

func TestBlacklist24Nodes(t *testing.T) {
	N := 15

	chain, nodeIDs := simpleChain(N)

	setLiarAndVictim(chain, "A", "K", 25)
	setLiarAndVictim(chain, "B", "K", 25)
	setLiarAndVictim(chain, "C", "K", 25)
	setLiarAndVictim(chain, "D", "K", 25)

	setLiarAndVictim(chain, "A", "G", 25)
	setLiarAndVictim(chain, "B", "G", 25)
	setLiarAndVictim(chain, "C", "G", 25)
	setLiarAndVictim(chain, "D", "G", 25)

	setLiarAndVictim(chain, "A", "H", 25)
	setLiarAndVictim(chain, "B", "H", 25)
	setLiarAndVictim(chain, "C", "H", 25)
	setLiarAndVictim(chain, "D", "H", 25)

	setLiarAndVictim(chain, "A", "I", 25)
	setLiarAndVictim(chain, "B", "I", 25)
	setLiarAndVictim(chain, "C", "I", 25)
	setLiarAndVictim(chain, "D", "I", 25)

	setLiarAndVictim(chain, "A", "J", 25)
	setLiarAndVictim(chain, "B", "J", 25)
	setLiarAndVictim(chain, "C", "J", 25)
	setLiarAndVictim(chain, "D", "J", 25)

	log.Print(checkBlacklistWithRemovedLatencies(chain, nodeIDs, false))

}
