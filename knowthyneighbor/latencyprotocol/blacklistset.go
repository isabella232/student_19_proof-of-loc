/*

Blacklistset is a set of public keys corresponding to blacklisted nodes, with the number of Strikes against them

*/

package latencyprotocol

import (
	//"go.dedis.ch/onet/v3/log"

	"sort"
	"strconv"

	sigAlg "golang.org/x/crypto/ed25519"
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

//AddWithStrikesStringKey adds a node's public key as a string to a blacklist a given number of times
func (set *Blacklistset) AddWithStrikesStringKey(key string, Strikes int) {
	_, isPresent := set.Strikes[key]
	if !isPresent {
		set.Strikes[key] = Strikes
	} else {
		set.Strikes[key] += Strikes
	}

}

//AddWithStrikes adds a node's public key to a blacklist a given number of times
func (set *Blacklistset) AddWithStrikes(key sigAlg.PublicKey, Strikes int) {
	set.AddWithStrikesStringKey(string(key), Strikes)
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

//ContainsAsString check if a node is blacklisted
func (set *Blacklistset) ContainsAsString(key string) bool {
	nbStrikes, isPresent := set.Strikes[string(key)]
	if !isPresent {
		return false
	}

	return nbStrikes > 0
}

//NumberStrikes give the numeber of Strikes a node got
func (set *Blacklistset) NumberStrikes(key sigAlg.PublicKey) int {
	return set.Strikes[string(key)]
}

//UpperThreshold returns the maximum number of strikes a victim node can get
func UpperThreshold(N int) int {
	third := float64(N) / 3
	return int(int(third)*int(N-1)) * 6
	//Multiply by 6 because that's how often a triangle will be tested
	//return (N / 3) * (N - (N / 3) - 1)

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
		if nbStrikes > 0 {
			size++
		}
	}
	return size
}

//IsEmpty returns whether a blacklist is empty
func (set *Blacklistset) IsEmpty() bool {
	return set.Size() <= 0
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

//NodesEqual checks if two sets have the same content
func (set *Blacklistset) NodesEqual(otherset *Blacklistset) bool {

	// If one is nil, the other must also be nil.
	if (set == nil) != (otherset == nil) {
		return false
	}

	if set.Size() != otherset.Size() {
		return false
	}

	for key, nbStrikes := range set.Strikes {
		if otherset.Strikes[key] != nbStrikes && (otherset.Strikes[key] == 0 || nbStrikes == 0) {
			return false
		}
	}

	return true
}

//ToString returns a string format of the Strikes
func (set *Blacklistset) ToString() string {
	if set.IsEmpty() {
		return "Blacklist empty\n"
	}

	keys := make([]string, 0, len(set.Strikes))
	for key := range set.Strikes {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	str := "\n"
	for _, key := range keys {
		val := set.Strikes[key]
		if val > 0 {
			str += key + ": " + strconv.Itoa(val) + "\n"
		}
	}
	return str
}

//ToStringFake returns a string format of the Strikes of an artificial network
func (set *Blacklistset) ToStringFake() string {
	if set.IsEmpty() {
		return "Blacklist empty\n"
	}

	str := "\n"
	for i := 0; i < len(set.Strikes); i++ {
		key := "N" + strconv.Itoa(i)
		val := set.Strikes[key]
		if val > 0 {
			str += key + ": " + strconv.Itoa(val) + "\n"
		}
	}
	return str
}

//NodesToString simply print which nodes are blacklisted
func (set *Blacklistset) NodesToString() string {

	if set.Size() == 0 {
		return "Blacklist Empty"
	}
	keys := make([]string, 0, len(set.Strikes))
	for key := range set.Strikes {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	str := ""
	for _, key := range keys {
		str += key + "-"
	}
	return str[:len(str)-1]
}

//PrintDifferencesTo returns a string showing the differences between two blacklistsets
func (set *Blacklistset) PrintDifferencesTo(other *Blacklistset) string {

	keys := make([]string, 0, len(set.Strikes))
	for key := range set.Strikes {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	str := "\n"
	for _, key := range keys {
		val1 := set.Strikes[key]
		val2, here := other.Strikes[key]
		if here {
			str += key + ": " + strconv.Itoa(val1) + " -> " + strconv.Itoa(val2) + "\n"
		} else {
			str += key + ": " + strconv.Itoa(val1) + " -> 0\n"
		}
	}
	return str
}

//CombineWith adds the content of another blacklist to the blacklists
func (set *Blacklistset) CombineWith(other *Blacklistset) {
	for k, v := range other.Strikes {
		set.AddWithStrikesStringKey(k, v)
	}
}

//NbStrikesOf returns the number of strikes of a given node
func (set *Blacklistset) NbStrikesOf(node string) int {
	strikes, exists := set.Strikes[node]
	if !exists {
		return 0
	}
	return strikes
}
