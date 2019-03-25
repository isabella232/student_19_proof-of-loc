package latencyprotocol

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
	DstNonce      Nonce
	Latency       time.Duration
	SignedLatency []byte
}

//SignedForeignLatency represents the latency a block signs for another block, using a timestamp to prevent the block from reusing our signature
type SignedForeignLatency struct {
	timestamp     time.Time
	signedLatency []byte
}

//PingMsg4 represents the content of the latency protocol's fourth message
type PingMsg4 struct {
	LocalLatency               time.Duration
	SignedLocalLatency         []byte
	SignedForeignLatency       SignedForeignLatency
	DoubleSignedForeignLatency []byte
}

//PingMsg5 represents the content of the latency protocol's fifth message
type PingMsg5 struct {
	SignedForeignLatency       SignedForeignLatency
	DoubleSignedForeignLatency []byte
}

func (Node *Node) sendMessage1(dstNodeID *NodeID) {

	nonce := Nonce(rand.Int())
	timestamp := time.Now()

	msgContent := &PingMsg1{
		SrcNonce:  nonce,
		Timestamp: timestamp,
	}

	_, alreadyStarted := Node.LatenciesInConstruction[string(dstNodeID.PublicKey)]

	if alreadyStarted {
		return
	}

	latConstr := LatencyConstructor{
		StartedLocally:    true,
		CurrentMsgNb:      1,
		DstID:             dstNodeID,
		Nonce:             nonce,
		LocalTimestamps:   make([]time.Time, 2),
		ForeignTimestamps: make([]time.Time, 2),
		ClockSkews:        make([]time.Duration, 2),
		Latency:           0,
	}

	latConstr.LocalTimestamps[0] = timestamp

	Node.LatenciesInConstruction[string(dstNodeID.PublicKey)] = &latConstr

	unsigned, err := protobuf.Encode(msgContent)
	if err != nil {
		//TODO
		return
	}

	signed := sigAlg.Sign(Node.PrivateKey, unsigned)

	msg := &PingMsg{
		Src:       *Node.ID.ServerID,
		Dst:       *dstNodeID.ServerID,
		SeqNb:     1,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsigned,
		SignedContent:   signed,
	}

	srcAddress := Node.ID.ServerID.Address.NetworkAddress()
	dstAddress := dstNodeID.ServerID.Address.NetworkAddress()

	SendMessage(msg, srcAddress, dstAddress)

}

func (Node *Node) checkMessage1(msg *PingMsg) (*PingMsg1, bool) {

	newPubKey := msg.PublicKey

	_, alreadyStarted := Node.LatenciesInConstruction[string(newPubKey)]

	if alreadyStarted {
		return nil, false
	}

	content := PingMsg1{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		return nil, false
	}

	sigCorrect := sigAlg.Verify(newPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		return nil, false
	}

	if !isFresh(content.Timestamp, freshnessDelta) {
		return nil, false
	}

	return &content, true

}

func (Node *Node) sendMessage2(msg *PingMsg, msgContent *PingMsg1) {

	nonce := Nonce(rand.Int())

	latencyConstr := &LatencyConstructor{
		StartedLocally:    false,
		CurrentMsgNb:      2,
		DstID:             &NodeID{&msg.Dst, msg.PublicKey},
		Nonce:             nonce,
		LocalTimestamps:   make([]time.Time, 2),
		ForeignTimestamps: make([]time.Time, 2),
		ClockSkews:        make([]time.Duration, 2),
		Latency:           0,
	}

	localtime := time.Now()
	latencyConstr.LocalTimestamps[0] = localtime
	latencyConstr.ForeignTimestamps[0] = msgContent.Timestamp
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

func (Node *Node) checkMessage2(msg *PingMsg) (*PingMsg2, bool) {

	senderPubKey := msg.PublicKey

	//check if we are building a latency for this
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[string(senderPubKey)]

	if !alreadyStarted {
		return nil, false
	}

	//extract content
	content := PingMsg2{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		return nil, false
	}

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		return nil, false
	}

	//check freshness
	if !isFresh(content.Timestamp, freshnessDelta) {
		return nil, false
	}

	//Check message 2 sent after message 1
	if content.Timestamp.Before(latencyConstr.LocalTimestamps[0]) {
		return nil, false
	}

	//check nonce
	if content.DstNonce != latencyConstr.Nonce {
		return nil, false
	}

	return &content, true

}

func (Node *Node) sendMessage3(msg *PingMsg, msgContent *PingMsg2) {

	latencyConstr := Node.LatenciesInConstruction[string(msg.PublicKey)]
	latencyConstr.CurrentMsgNb += 2

	localtime := time.Now()
	latencyConstr.LocalTimestamps[1] = localtime
	latencyConstr.ForeignTimestamps[0] = msgContent.Timestamp
	latencyConstr.ClockSkews[0] = localtime.Sub(msgContent.Timestamp)

	latency := localtime.Sub(latencyConstr.LocalTimestamps[0])
	latencyConstr.Latency = latency
	unsignedLatency, err := protobuf.Encode(latency)
	signedLatency := sigAlg.Sign(Node.PrivateKey, unsignedLatency)

	msg3Content := &PingMsg3{
		DstNonce:      msgContent.SrcNonce,
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
		Src:             msg.Dst,
		Dst:             msg.Src,
		SeqNb:           3,
		PublicKey:       Node.ID.PublicKey,
		UnsignedContent: unsignedContent,
		SignedContent:   signedContent,
	}

	srcAddress := Node.ID.ServerID.Address.NetworkAddress()
	dstAddress := msg.Dst.Address.NetworkAddress()

	SendMessage(newMsg, srcAddress, dstAddress)

}

func (Node *Node) checkMessage3(msg *PingMsg) (*PingMsg3, bool) {

	senderPubKey := msg.PublicKey

	//check if we are building a latency for this
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[string(senderPubKey)]

	if !alreadyStarted {
		return nil, false
	}

	//extract content
	content := PingMsg3{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		return nil, false
	}

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		return nil, false
	}

	sigTimestamp := latencyConstr.ForeignTimestamps[0].Add(content.Latency)
	latencyConstr.ForeignTimestamps[1] = sigTimestamp

	//check freshness
	if !isFresh(sigTimestamp, freshnessDelta) {
		return nil, false
	}

	//check nonce
	if content.DstNonce != latencyConstr.Nonce {
		return nil, false
	}

	return &content, true

}

func (Node *Node) sendMessage4(msg *PingMsg, msgContent *PingMsg3) {

	latencyConstr := Node.LatenciesInConstruction[string(msg.PublicKey)]
	latencyConstr.CurrentMsgNb += 2

	localtime := time.Now()
	latencyConstr.LocalTimestamps[1] = localtime
	latencyConstr.ClockSkews[1] = localtime.Sub(latencyConstr.ForeignTimestamps[1])

	if !acceptableDifference(latencyConstr.ClockSkews[0], latencyConstr.ClockSkews[1], intervallDelta) {
		//TODO
		return
	}

	localLatency := localtime.Sub(latencyConstr.LocalTimestamps[0])

	latencyConstr.Latency = localLatency

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

func (Node *Node) checkMessage4(msg *PingMsg) (*PingMsg4, bool) {

	senderPubKey := msg.PublicKey

	//check if we are building a latency for this
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[string(senderPubKey)]

	if !alreadyStarted {
		return nil, false
	}

	//extract content
	content := PingMsg4{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		return nil, false
	}

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		return nil, false
	}

	sentTimestamp := content.SignedForeignLatency.timestamp
	latencyConstr.ForeignTimestamps[1] = sentTimestamp

	//check freshness
	if !isFresh(sentTimestamp, freshnessDelta) {
		return nil, false
	}

	return &content, true

}

func (Node *Node) sendMessage5(msg *PingMsg, msgContent *PingMsg4) *ConfirmedLatency {

	latencyConstr := Node.LatenciesInConstruction[string(msg.PublicKey)]
	latencyConstr.CurrentMsgNb += 2

	localtime := time.Now()
	latencyConstr.LocalTimestamps[1] = localtime
	latencyConstr.ClockSkews[1] = localtime.Sub(latencyConstr.ForeignTimestamps[1])

	if !acceptableDifference(latencyConstr.ClockSkews[0], latencyConstr.ClockSkews[1], intervallDelta) {
		//TODO
		return nil
	}

	if !acceptableDifference(latencyConstr.Latency, msgContent.LocalLatency, intervallDelta) {
		// TODO + account for clock skew
		return nil
	}

	signedForeignLatency := SignedForeignLatency{localtime, msgContent.SignedLocalLatency}
	signedForeignLatencyBytes, err := protobuf.Encode(signedForeignLatency)
	if err != nil {
		return nil
	}

	doubleSignedforeignLatency := sigAlg.Sign(Node.PrivateKey, signedForeignLatencyBytes)

	msg5Content := &PingMsg5{
		SignedForeignLatency:       signedForeignLatency,
		DoubleSignedForeignLatency: doubleSignedforeignLatency,
	}

	unsignedContent, err := protobuf.Encode(msg5Content)
	if err != nil {
		//TODO
		return nil
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

	newLatency := &ConfirmedLatency{
		Latency:            latencyConstr.Latency,
		Timestamp:          latencyConstr.ForeignTimestamps[1],
		SignedConfirmation: msgContent.DoubleSignedForeignLatency,
	}

	return newLatency

}

func (Node *Node) checkMessage5(msg *PingMsg) (*ConfirmedLatency, bool) {

	senderPubKey := msg.PublicKey

	//check if we are building a latency for this
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[string(senderPubKey)]

	if !alreadyStarted {
		return nil, false
	}

	//extract content
	content := PingMsg5{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		return nil, false
	}

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		return nil, false
	}

	sentTimestamp := content.SignedForeignLatency.timestamp

	//check freshness
	if !isFresh(sentTimestamp, freshnessDelta) {
		return nil, false
	}

	//Check message 5 sent after message 4
	if sentTimestamp.Before(latencyConstr.LocalTimestamps[1]) {
		return nil, false
	}

	newLatency := &ConfirmedLatency{
		Latency:            latencyConstr.Latency,
		Timestamp:          sentTimestamp,
		SignedConfirmation: content.DoubleSignedForeignLatency,
	}

	return newLatency, true

}

func isFresh(timestamp time.Time, delta time.Duration) bool {
	return timestamp.After(time.Now().Add(-delta))
}

func acceptableDifference(time1 time.Duration, time2 time.Duration, delta time.Duration) bool {
	return time1-time2 < delta && time2-time1 < delta
}
