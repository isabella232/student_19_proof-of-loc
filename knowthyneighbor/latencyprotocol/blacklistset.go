package latencyprotocol

import (
	sigAlg "golang.org/x/crypto/ed25519"
)

//Blacklistset is a set of public keys corresponding to blacklisted nodes
type Blacklistset struct {
	set map[string]bool
}

//NewBlacklistset constructs a new blacklistset
func NewBlacklistset() Blacklistset {
	return Blacklistset{
		make(map[string]bool, 0),
	}
}

//Add adds a node's public key to a blacklist
func (set *Blacklistset) Add(key sigAlg.PublicKey) {
	set.set[string(key)] = true
}

//Remove removes a node's public key to a blacklist
func (set *Blacklistset) Remove(key sigAlg.PublicKey) {
	set.set[string(key)] = false
}

//Contains check if a node is balcklisted
func (set *Blacklistset) Contains(key sigAlg.PublicKey) bool {
	return set.set[string(key)]
}

//Size returns the size of the set
func (set *Blacklistset) Size() int {
	return len(set.set)
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

	for key := range set.set {
		if otherset.set[key] == false {
			return false
		}
	}

	return true
}
