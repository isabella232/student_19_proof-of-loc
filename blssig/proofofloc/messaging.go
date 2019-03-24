package proofofloc

import (
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
	sigAlg "golang.org/x/crypto/ed25519"
	"math/rand"
	"time"
)

const freshnessDelta = 10 * time.Second
const intervallDelta = 10 * time.Second

//PingMsg represents a message sent to another validator
type PingMsg struct {
	Src       network.ServerIdentity
	Dst       network.ServerIdentity
	SeqNb     int
	PublicKey sigAlg.PublicKey

	UnsignedContent []byte
	SignedContent   []byte
}

//PingMsg1 represents the content of the latency protocol's first message
type PingMsg1 struct {
	SrcNonce  Nonce
	Timestamp time.Time
}

//PingMsg2 represents the content of the latency protocol's second message
type PingMsg2 struct {
	SrcNonce  Nonce
	DstNonce  Nonce
	Timestamp time.Time
}

//PingMsg3 represents the content of the latency protocol's third message
type PingMsg3 struct {
	SrcNonce      Nonce
	DstNonce      Nonce
	Timestamp     time.Time
	Latency       time.Duration
	SignedLatency []byte
}

type SignedForeignLatency struct {
	timestamp     time.Time
	signedLatency []byte
}

//PingMsg4 represents the content of the latency protocol's fourth message
type PingMsg4 struct {
	SrcNonce                   Nonce
	DstNonce                   Nonce
	Timestamp                  time.Time
	LocalLatency               time.Duration
	SignedLocalLatency         []byte
	SignedForeignLatency       SignedForeignLatency
	DoubleSignedForeignLatency []byte
}

//PingMsg5 represents the content of the latency protocol's fifth message
type PingMsg5 struct {
	DstNonce                   Nonce
	Timestamp                  time.Time
	SignedForeignLatency       SignedForeignLatency
	DoubleSignedForeignLatency []byte
}

func (Node *Node) sendMessage1(dstNode *Node) {

	nonce := Nonce(rand.Int())
	timestamp := time.Now()

	msgContent := &PingMsg1{
		SrcNonce:  nonce,
		Timestamp: timestamp,
	}

	_, alreadyStarted := Node.LatenciesInConstruction[string(dstNode.ID.PublicKey)]

	if alreadyStarted {
		return
	}

	latConstr := LatencyConstructor{
		StartedLocally: true,
		CurrentMsgNb:   1,
		DstID:          dstNode.ID,
		Nonces:         make([]Nonce, 2),
		Timestamps:     make([]time.Time, 2),
		ClockSkews:     make([]time.Duration, 2),
		Latency:        0,
	}

	latConstr.Nonces[0] = nonce
	latConstr.Timestamps[0] = timestamp

	Node.LatenciesInConstruction[string(dstNode.ID.PublicKey)] = &latConstr

	unsigned, err := protobuf.Encode(msgContent)
	if err != nil {
		//TODO
		return
	}

	signed := sigAlg.Sign(Node.PrivateKey, unsigned)

	msg := &PingMsg{
		Src:       *Node.ID.ServerID,
		Dst:       *dstNode.ID.ServerID,
		SeqNb:     1,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsigned,
		SignedContent:   signed,
	}

	srcAddress := Node.ID.ServerID.Address.NetworkAddress()
	dstAddress := dstNode.ID.ServerID.Address.NetworkAddress()

	SendMessage(msg, srcAddress, dstAddress)

}

func (Node *Node) checkMessage1(msg PingMsg) bool {

	newPubKey := msg.PublicKey

	_, alreadyStarted := Node.LatenciesInConstruction[string(newPubKey)]

	if alreadyStarted {
		return false
	}

	content := PingMsg1{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		return false
	}

	sigCorrect := sigAlg.Verify(newPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		return false
	}

	if !isFresh(content.Timestamp, freshnessDelta) {
		return false
	}

	return true

}

func (Node *Node) sendMessage2(msg PingMsg, msgContent PingMsg1) {

	latencyConstr := &LatencyConstructor{
		StartedLocally: false,
		CurrentMsgNb:   2,
		DstID:          &NodeID{&msg.Dst, msg.PublicKey},
		Nonces:         make([]Nonce, 2),
		Timestamps:     make([]time.Time, 2),
		ClockSkews:     make([]time.Duration, 2),
		Latency:        0,
	}

	nonce := Nonce(rand.Int())
	latencyConstr.Nonces[0] = nonce

	localtime := time.Now()
	latencyConstr.Timestamps[0] = localtime
	latencyConstr.ClockSkews[0] = localtime.Sub(msgContent.Timestamp)

	msg2Content := &PingMsg2{
		SrcNonce:  nonce,
		DstNonce:  msgContent.SrcNonce,
		Timestamp: localtime,
	}

	unsigned, err := protobuf.Encode(msg2Content)
	if err != nil {
		//TODO
		return
	}

	signed := sigAlg.Sign(Node.PrivateKey, unsigned)

	newMsg := &PingMsg{
		Src:       msg.Dst,
		Dst:       msg.Src,
		SeqNb:     2,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsigned,
		SignedContent:   signed,
	}

	srcAddress := Node.ID.ServerID.Address.NetworkAddress()
	dstAddress := msg.Dst.Address.NetworkAddress()

	SendMessage(newMsg, srcAddress, dstAddress)

}

func (Node *Node) checkMessage2(msg PingMsg) bool {

	senderPubKey := msg.PublicKey

	//check if we are building a latency for this
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[string(senderPubKey)]

	if !alreadyStarted {
		return false
	}

	//extract content
	content := PingMsg2{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		return false
	}

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		return false
	}

	//check freshness
	if !isFresh(content.Timestamp, freshnessDelta) {
		return false
	}

	//Check message 2 sent after message 1
	if content.Timestamp.Before(latencyConstr.Timestamps[0]) {
		return false
	}

	//check nonce
	if content.DstNonce != latencyConstr.Nonces[0] {
		return false
	}

	return true

}

func (Node *Node) sendMessage3(msg PingMsg, msgContent PingMsg2) {

	latencyConstr := Node.LatenciesInConstruction[string(msg.PublicKey)]
	latencyConstr.CurrentMsgNb += 2

	nonceToSend := Nonce(rand.Int())
	latencyConstr.Nonces[1] = nonceToSend

	localtime := time.Now()
	latencyConstr.Timestamps[1] = localtime
	latencyConstr.ClockSkews[0] = localtime.Sub(msgContent.Timestamp)

	latency := localtime.Sub(latencyConstr.Timestamps[0])
	unsignedLatency, err := protobuf.Encode(latency)
	signedLatency := sigAlg.Sign(Node.PrivateKey, unsignedLatency)

	msg3Content := &PingMsg3{
		SrcNonce:      nonceToSend,
		DstNonce:      msgContent.SrcNonce,
		Timestamp:     localtime,
		Latency:       latency,
		SignedLatency: signedLatency,
	}

	unsignedContent, err := protobuf.Encode(msg3Content)
	if err != nil {
		//TODO
		return
	}

	signedContent := sigAlg.Sign(Node.PrivateKey, unsignedContent)

	newMsg := &PingMsg{
		Src:       msg.Dst,
		Dst:       msg.Src,
		SeqNb:     3,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsignedContent,
		SignedContent:   signedContent,
	}

	srcAddress := Node.ID.ServerID.Address.NetworkAddress()
	dstAddress := msg.Dst.Address.NetworkAddress()

	SendMessage(newMsg, srcAddress, dstAddress)

}

func (Node *Node) checkMessage3(msg PingMsg) bool {

	senderPubKey := msg.PublicKey

	//check if we are building a latency for this
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[string(senderPubKey)]

	if !alreadyStarted {
		return false
	}

	//extract content
	content := PingMsg3{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		return false
	}

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		return false
	}

	//check freshness
	if !isFresh(content.Timestamp, freshnessDelta) {
		return false
	}

	//Check message 3 sent after message 2
	if content.Timestamp.Before(latencyConstr.Timestamps[0]) {
		return false
	}

	//check nonce
	if content.DstNonce != latencyConstr.Nonces[0] {
		return false
	}

	return true

}

func (Node *Node) sendMessage4(msg PingMsg, msgContent PingMsg3) {

	latencyConstr := Node.LatenciesInConstruction[string(msg.PublicKey)]
	latencyConstr.CurrentMsgNb += 2

	nonceToSend := Nonce(rand.Int())
	latencyConstr.Nonces[1] = nonceToSend

	localtime := time.Now()
	latencyConstr.Timestamps[1] = localtime
	latencyConstr.ClockSkews[1] = localtime.Sub(msgContent.Timestamp)

	if !acceptableDifference(latencyConstr.ClockSkews[0], latencyConstr.ClockSkews[1], intervallDelta) {
		//TODO
		return
	}

	localLatency := localtime.Sub(latencyConstr.Timestamps[0])

	if !acceptableDifference(localLatency, msgContent.Latency, intervallDelta) {
		// TODO + account for clock skew
		return
	}

	unsignedLocalLatency, err := protobuf.Encode(localLatency)
	if err != nil {
		return
	}
	signedLocalLatency := sigAlg.Sign(Node.PrivateKey, unsignedLocalLatency)

	signedForeignLatency := SignedForeignLatency{localtime, msgContent.SignedLatency}
	signedForeignLatencyBytes, err := protobuf.Encode(signedForeignLatency)
	if err != nil {
		return
	}

	doubleSignedforeignLatency := sigAlg.Sign(Node.PrivateKey, signedForeignLatencyBytes)

	msg4Content := &PingMsg4{
		SrcNonce:                   nonceToSend,
		DstNonce:                   msgContent.SrcNonce,
		Timestamp:                  localtime,
		LocalLatency:               localLatency,
		SignedLocalLatency:         signedLocalLatency,
		SignedForeignLatency:       signedForeignLatency,
		DoubleSignedForeignLatency: doubleSignedforeignLatency,
	}

	unsignedContent, err := protobuf.Encode(msg4Content)
	if err != nil {
		//TODO
		return
	}

	signedContent := sigAlg.Sign(Node.PrivateKey, unsignedContent)

	newMsg := &PingMsg{
		Src:       msg.Dst,
		Dst:       msg.Src,
		SeqNb:     4,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsignedContent,
		SignedContent:   signedContent,
	}

	srcAddress := Node.ID.ServerID.Address.NetworkAddress()
	dstAddress := msg.Dst.Address.NetworkAddress()

	SendMessage(newMsg, srcAddress, dstAddress)

}

func (Node *Node) checkMessage4(msg PingMsg) bool {

	senderPubKey := msg.PublicKey

	//check if we are building a latency for this
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[string(senderPubKey)]

	if !alreadyStarted {
		return false
	}

	//extract content
	content := PingMsg4{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		return false
	}

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		return false
	}

	//check freshness
	if !isFresh(content.Timestamp, freshnessDelta) {
		return false
	}

	//Check message 4 sent after message 3
	if content.Timestamp.Before(latencyConstr.Timestamps[1]) {
		return false
	}

	//check nonce
	if content.DstNonce != latencyConstr.Nonces[1] {
		return false
	}

	return true

}

func (Node *Node) sendMessage5(msg PingMsg, msgContent PingMsg4) {

	latencyConstr := Node.LatenciesInConstruction[string(msg.PublicKey)]
	latencyConstr.CurrentMsgNb += 2

	localtime := time.Now()
	latencyConstr.Timestamps[1] = localtime
	latencyConstr.ClockSkews[1] = localtime.Sub(msgContent.Timestamp)

	if !acceptableDifference(latencyConstr.ClockSkews[0], latencyConstr.ClockSkews[1], intervallDelta) {
		//TODO
		return
	}

	if !acceptableDifference(latencyConstr.Latency, msgContent.LocalLatency, intervallDelta) {
		// TODO + account for clock skew
		return
	}

	signedForeignLatency := SignedForeignLatency{localtime, msgContent.SignedLocalLatency}
	signedForeignLatencyBytes, err := protobuf.Encode(signedForeignLatency)
	if err != nil {
		return
	}

	doubleSignedforeignLatency := sigAlg.Sign(Node.PrivateKey, signedForeignLatencyBytes)

	msg5Content := &PingMsg5{
		DstNonce:                   msgContent.SrcNonce,
		Timestamp:                  localtime,
		SignedForeignLatency:       signedForeignLatency,
		DoubleSignedForeignLatency: doubleSignedforeignLatency,
	}

	unsignedContent, err := protobuf.Encode(msg5Content)
	if err != nil {
		//TODO
		return
	}

	signedContent := sigAlg.Sign(Node.PrivateKey, unsignedContent)

	newMsg := &PingMsg{
		Src:       msg.Dst,
		Dst:       msg.Src,
		SeqNb:     5,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsignedContent,
		SignedContent:   signedContent,
	}

	srcAddress := Node.ID.ServerID.Address.NetworkAddress()
	dstAddress := msg.Dst.Address.NetworkAddress()

	SendMessage(newMsg, srcAddress, dstAddress)

	//Add msgContent.Doublesigned to block

}

func (Node *Node) checkMessage5(msg PingMsg) bool {

	senderPubKey := msg.PublicKey

	//check if we are building a latency for this
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[string(senderPubKey)]

	if !alreadyStarted {
		return false
	}

	//extract content
	content := PingMsg5{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		return false
	}

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		return false
	}

	//check freshness
	if !isFresh(content.Timestamp, freshnessDelta) {
		return false
	}

	//Check message 5 sent after message 4
	if content.Timestamp.Before(latencyConstr.Timestamps[1]) {
		return false
	}

	//check nonce
	if content.DstNonce != latencyConstr.Nonces[1] {
		return false
	}

	return true

}

func isFresh(timestamp time.Time, delta time.Duration) bool {
	return timestamp.After(time.Now().Add(-delta))
}

func acceptableDifference(time1 time.Duration, time2 time.Duration, delta time.Duration) bool {
	return time1-time2 < delta && time2-time1 < delta
}
