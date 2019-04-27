package latencyprotocol

import (
	sigAlg "golang.org/x/crypto/ed25519"
	"strconv"
)

//Blacklistset is a set of public keys corresponding to blacklisted nodes, with the number of Strikes against them
type Blacklistset struct {
	Strikes map[string]int
}

//NewBlacklistset constructs a new blacklistset
func NewBlacklistset() Blacklistset {
	return Blacklistset{
		make(map[string]int, 0),
	}
}

//Add adds a node's public key to a blacklist
func (set *Blacklistset) Add(key sigAlg.PublicKey) {
	_, isPresent := set.Strikes[string(key)]
	if !isPresent {
		set.Strikes[string(key)] = 1
	} else {
		set.Strikes[string(key)]++
	}

}

//AddWithStrikes adds a node's public key to a blacklist a given number of times
func (set *Blacklistset) AddWithStrikes(key sigAlg.PublicKey, Strikes int) {
	_, isPresent := set.Strikes[string(key)]
	if !isPresent {
		set.Strikes[string(key)] = Strikes
	} else {
		set.Strikes[string(key)] += Strikes
	}

}

//Remove removes a node's public key to a blacklist
func (set *Blacklistset) Remove(key sigAlg.PublicKey) {
	set.Strikes[string(key)] = 0
}

//Contains check if a node is balcklisted
func (set *Blacklistset) Contains(key sigAlg.PublicKey, thresh int) bool {
	nbStrikes, isPresent := set.Strikes[string(key)]
	if !isPresent {
		return false
	}

	return nbStrikes > thresh
}

//NumberStrikes give the numeber of Strikes a node got
func (set *Blacklistset) NumberStrikes(key sigAlg.PublicKey) int {
	return set.Strikes[string(key)]
}

//GetBlacklistWithThreshold returns a new blacklist containing only the nodes with more than a given threshold of Strikes
func (set *Blacklistset) GetBlacklistWithThreshold(thresh int) Blacklistset {
	threshBlacklist := NewBlacklistset()
	for key, Strikes := range set.Strikes {
		if Strikes > thresh {
			threshBlacklist.AddWithStrikes(sigAlg.PublicKey([]byte(key)), Strikes)
		}
	}
	return threshBlacklist
}

//Size returns the size of the set
func (set *Blacklistset) Size() int {
	size := 0
	for _, nbStrikes := range set.Strikes {
		if nbStrikes != 0 {
			size++
		}
	}
	return size
}

//Equals checks if two sets have the same content
func (set *Blacklistset) Equals(otherset *Blacklistset) bool {

	// If one is nil, the other must also be nil.
	if (set == nil) != (otherset == nil) {
		return false
	}

	if set.Size() != otherset.Size() {
		return false
	}

	for key, nbStrikes := range set.Strikes {
		if otherset.Strikes[key] != nbStrikes {
			return false
		}
	}

	return true
}

//ToString returns a string format of the Strikes
func (set *Blacklistset) ToString() string {
	str := "\n"
	for key, val := range set.Strikes {
		str += key + ": " + strconv.Itoa(val) + "\n"
	}
	return str
}

//PrintDifferencesTo returns a string showing the differences between two blacklistsets
func (set *Blacklistset) PrintDifferencesTo(other *Blacklistset) string {
	str := "\n"
	for key, val1 := range set.Strikes {
		val2, here := other.Strikes[key]
		if here {
			str += key + ": " + strconv.Itoa(val1) + " -> " + strconv.Itoa(val2) + "\n"
		}
	}

	return str
}
