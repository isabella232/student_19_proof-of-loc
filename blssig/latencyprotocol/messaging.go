package latencyprotocol

import (
	"encoding/base64"
	"errors"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
	sigAlg "golang.org/x/crypto/ed25519"
	"math"
	"math/rand"
	"strconv"
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
	Timestamp     time.Time
	SignedLatency []byte
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

func (Node *Node) sendMessage1(dstNodeID *NodeID) error {
	log.LLvl1("Sending message 1")

	nonce := Nonce(rand.Intn(math.MaxInt32))

	timestamp := time.Now()

	msgContent := &PingMsg1{
		SrcNonce:  nonce,
		Timestamp: timestamp,
	}

	encodedKey := base64.StdEncoding.EncodeToString(dstNodeID.PublicKey)
	_, alreadyStarted := Node.LatenciesInConstruction[encodedKey]

	if alreadyStarted {
		return errors.New("Already started messaging this node")
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

	Node.LatenciesInConstruction[encodedKey] = &latConstr

	unsigned, err := protobuf.Encode(msgContent)
	if err != nil {
		return err
	}

	signed := sigAlg.Sign(Node.PrivateKey, unsigned)

	msg := PingMsg{
		Src:       *Node.ID.ServerID,
		Dst:       *dstNodeID.ServerID,
		SeqNb:     1,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsigned,
		SignedContent:   signed,
	}

	srcAddress := Node.SendingAddress.NetworkAddress()
	dstAddress := dstNodeID.ServerID.Address.NetworkAddress()

	log.LLvl1("Sending message 1 from " + srcAddress + " to " + dstAddress)

	err = SendMessage(msg, srcAddress, dstAddress)
	if err != nil {
		return err
	}

	return nil

}

func (Node *Node) checkMessage1(msg *PingMsg) (*PingMsg1, bool) {
	log.LLvl1("Checking message 1")

	newPubKey := msg.PublicKey

	sigCorrect := sigAlg.Verify(newPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		return nil, false
	}

	encodedKey := base64.StdEncoding.EncodeToString(newPubKey)
	_, alreadyStarted := Node.LatenciesInConstruction[encodedKey]

	if alreadyStarted {
		return nil, false
	}

	content := PingMsg1{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		return nil, false
	}

	if !isFresh(content.Timestamp, freshnessDelta) {
		return nil, false
	}

	return &content, true

}

func (Node *Node) sendMessage2(msg *PingMsg, msgContent *PingMsg1) {
	log.LLvl1("Sending message 2")

	nonce := Nonce(rand.Intn(math.MaxInt32))

	latencyConstr := LatencyConstructor{
		StartedLocally:    false,
		CurrentMsgNb:      2,
		DstID:             &NodeID{&msg.Src, msg.PublicKey},
		Nonce:             nonce,
		LocalTimestamps:   make([]time.Time, 2),
		ForeignTimestamps: make([]time.Time, 2),
		ClockSkews:        make([]time.Duration, 2),
		Latency:           0,
	}

	encodedKey := base64.StdEncoding.EncodeToString(msg.PublicKey)
	Node.LatenciesInConstruction[encodedKey] = &latencyConstr

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
		log.LLvl1(err)
		return
	}

	signed := sigAlg.Sign(Node.PrivateKey, unsigned)

	newMsg := PingMsg{
		Src:       msg.Dst,
		Dst:       msg.Src,
		SeqNb:     2,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsigned,
		SignedContent:   signed,
	}

	srcAddress := Node.SendingAddress.NetworkAddress()
	dstAddress := msg.Src.Address.NetworkAddress()

	log.LLvl1("Sending message 2 from " + srcAddress + " to " + dstAddress)

	SendMessage(newMsg, srcAddress, dstAddress)

}

func (Node *Node) checkMessage2(msg *PingMsg) (*PingMsg2, bool) {
	log.LLvl1("Checking message 2")

	senderPubKey := msg.PublicKey

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		log.LLvl1("Signature incorrect")
		return nil, false
	}

	//check if we are building a latency for this
	encodedKey := base64.StdEncoding.EncodeToString(senderPubKey)
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[encodedKey]

	if !alreadyStarted {
		log.LLvl1("Not started yet")
		return nil, false
	}

	//extract content
	content := PingMsg2{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		log.LLvl1(err)
		return nil, false
	}

	//check freshness
	if !isFresh(content.Timestamp, freshnessDelta) {
		log.LLvl1("Not fresh enough")
		return nil, false
	}

	//Check message 2 sent after message 1
	if content.Timestamp.Before(latencyConstr.LocalTimestamps[0]) {
		log.LLvl1("Timestamp order wrong")
		return nil, false
	}

	//check nonce
	if content.DstNonce != latencyConstr.Nonce {
		log.LLvl1("Wrong nonce - Local: " + strconv.Itoa(int(latencyConstr.Nonce)))
		log.LLvl1("Sent: " + strconv.Itoa(int(content.DstNonce)))
		return nil, false
	}

	return &content, true

}

func (Node *Node) sendMessage3(msg *PingMsg, msgContent *PingMsg2) {
	log.LLvl1("Sending message 3")

	encodedKey := base64.StdEncoding.EncodeToString(msg.PublicKey)
	latencyConstr := Node.LatenciesInConstruction[encodedKey]
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
		log.LLvl1(err)
		return
	}

	signedContent := sigAlg.Sign(Node.PrivateKey, unsignedContent)

	newMsg := PingMsg{
		Src:             msg.Dst,
		Dst:             msg.Src,
		SeqNb:           3,
		PublicKey:       Node.ID.PublicKey,
		UnsignedContent: unsignedContent,
		SignedContent:   signedContent,
	}

	srcAddress := Node.SendingAddress.NetworkAddress()
	dstAddress := msg.Src.Address.NetworkAddress()

	log.LLvl1("Sending message 3 from " + srcAddress + " to " + dstAddress)

	SendMessage(newMsg, srcAddress, dstAddress)

}

func (Node *Node) checkMessage3(msg *PingMsg) (*PingMsg3, bool) {

	log.LLvl1("Checking message 3")

	senderPubKey := msg.PublicKey

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		log.LLvl1("Signature incorrect")
		return nil, false
	}

	//check if we are building a latency for this
	encodedKey := base64.StdEncoding.EncodeToString(senderPubKey)
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[encodedKey]

	if !alreadyStarted {
		log.LLvl1("Not started yet")
		return nil, false
	}

	//extract content
	content := PingMsg3{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		log.LLvl1(err)
		return nil, false
	}

	sigTimestamp := latencyConstr.ForeignTimestamps[0].Add(content.Latency)
	latencyConstr.ForeignTimestamps[1] = sigTimestamp

	//check freshness
	if !isFresh(sigTimestamp, freshnessDelta) {
		log.LLvl1("Old message")
		return nil, false
	}

	//check nonce
	if content.DstNonce != latencyConstr.Nonce {
		log.LLvl1("Nonce wrong")
		return nil, false
	}

	return &content, true

}

func (Node *Node) sendMessage4(msg *PingMsg, msgContent *PingMsg3) {
	log.LLvl1("Sending message 4")

	encodedKey := base64.StdEncoding.EncodeToString(msg.PublicKey)
	latencyConstr := Node.LatenciesInConstruction[encodedKey]
	latencyConstr.CurrentMsgNb += 2

	localtime := time.Now()
	latencyConstr.LocalTimestamps[1] = localtime
	latencyConstr.ClockSkews[1] = localtime.Sub(latencyConstr.ForeignTimestamps[1])

	if !acceptableDifference(latencyConstr.ClockSkews[0], latencyConstr.ClockSkews[1], intervallDelta) {
		log.LLvl1("Clock Skews too different")
		return
	}

	localLatency := localtime.Sub(latencyConstr.LocalTimestamps[0])

	latencyConstr.Latency = localLatency

	if !acceptableDifference(localLatency, msgContent.Latency, intervallDelta) {
		log.LLvl1("Latencies too different")
		return
	}

	unsignedLocalLatency, err := protobuf.Encode(&LatencyWrapper{localLatency})
	if err != nil {
		log.LLvl1(err)
		return
	}
	signedLocalLatency := sigAlg.Sign(Node.PrivateKey, unsignedLocalLatency)

	signedForeignLatency := SignedForeignLatency{localtime, msgContent.SignedLatency}
	signedForeignLatencyBytes, err := protobuf.Encode(&signedForeignLatency)
	if err != nil {
		log.LLvl1(err)
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
		log.LLvl1(err)
		return
	}

	signedContent := sigAlg.Sign(Node.PrivateKey, unsignedContent)

	newMsg := PingMsg{
		Src:       msg.Dst,
		Dst:       msg.Src,
		SeqNb:     4,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsignedContent,
		SignedContent:   signedContent,
	}

	srcAddress := Node.SendingAddress.NetworkAddress()
	dstAddress := msg.Src.Address.NetworkAddress()

	log.LLvl1("Sending message 4 from " + srcAddress + " to " + dstAddress)

	SendMessage(newMsg, srcAddress, dstAddress)

}

func (Node *Node) checkMessage4(msg *PingMsg) (*PingMsg4, *ConfirmedLatency, bool) {
	log.LLvl1("Checking message 4")

	senderPubKey := msg.PublicKey

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		log.LLvl1("Signature incorrest")
		return nil, nil, false
	}

	//check if we are building a latency for this
	encodedKey := base64.StdEncoding.EncodeToString(msg.PublicKey)
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[encodedKey]

	if !alreadyStarted {
		log.LLvl1("Not started yet")
		return nil, nil, false
	}

	//extract content
	content := PingMsg4{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		log.LLvl1(err)
		return nil, nil, false
	}

	sentTimestamp := content.SignedForeignLatency.Timestamp
	latencyConstr.ForeignTimestamps[1] = sentTimestamp

	//check freshness
	if !isFresh(sentTimestamp, freshnessDelta) {
		log.LLvl1("Not fresh enough")
		return nil, nil, false
	}

	newLatency := &ConfirmedLatency{
		Latency:            latencyConstr.Latency,
		Timestamp:          sentTimestamp,
		SignedConfirmation: content.DoubleSignedForeignLatency,
	}

	log.LLvl1("Returning new latency from message 5")

	return &content, newLatency, true

}

func (Node *Node) sendMessage5(msg *PingMsg, msgContent *PingMsg4) *ConfirmedLatency {
	log.LLvl1("Sending message 5")

	encodedKey := base64.StdEncoding.EncodeToString(msg.PublicKey)
	latencyConstr := Node.LatenciesInConstruction[encodedKey]
	latencyConstr.CurrentMsgNb += 2

	localtime := time.Now()
	latencyConstr.LocalTimestamps[1] = localtime
	latencyConstr.ClockSkews[1] = localtime.Sub(latencyConstr.ForeignTimestamps[1])

	if !acceptableDifference(latencyConstr.ClockSkews[0], latencyConstr.ClockSkews[1], intervallDelta) {
		log.LLvl1("Clock Skews too different")
		return nil
	}

	if !acceptableDifference(latencyConstr.Latency, msgContent.LocalLatency, intervallDelta) {
		log.LLvl1("Latencies too different")
		return nil
	}

	signedForeignLatency := SignedForeignLatency{localtime, msgContent.SignedLocalLatency}
	signedForeignLatencyBytes, err := protobuf.Encode(&signedForeignLatency)
	if err != nil {
		log.LLvl1(err)
		return nil
	}

	doubleSignedforeignLatency := sigAlg.Sign(Node.PrivateKey, signedForeignLatencyBytes)

	msg5Content := &PingMsg5{
		SignedForeignLatency:       signedForeignLatency,
		DoubleSignedForeignLatency: doubleSignedforeignLatency,
	}

	unsignedContent, err := protobuf.Encode(msg5Content)
	if err != nil {
		log.LLvl1(err)
		return nil
	}

	signedContent := sigAlg.Sign(Node.PrivateKey, unsignedContent)

	newMsg := PingMsg{
		Src:       msg.Dst,
		Dst:       msg.Src,
		SeqNb:     5,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsignedContent,
		SignedContent:   signedContent,
	}

	srcAddress := Node.SendingAddress.NetworkAddress()
	dstAddress := msg.Src.Address.NetworkAddress()

	log.LLvl1("Sending message 5 from " + srcAddress + " to " + dstAddress)

	SendMessage(newMsg, srcAddress, dstAddress)

	newLatency := &ConfirmedLatency{
		Latency:            latencyConstr.Latency,
		Timestamp:          latencyConstr.ForeignTimestamps[1],
		SignedConfirmation: msgContent.DoubleSignedForeignLatency,
	}

	return newLatency

}

func (Node *Node) checkMessage5(msg *PingMsg) (*ConfirmedLatency, bool) {
	log.LLvl1("Checking message 5")

	senderPubKey := msg.PublicKey

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		log.LLvl1("Signature incorrect")
		return nil, false
	}

	//check if we are building a latency for this
	encodedKey := base64.StdEncoding.EncodeToString(msg.PublicKey)
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[encodedKey]

	if !alreadyStarted {
		log.LLvl1("Already started")
		return nil, false
	}

	//extract content
	content := PingMsg5{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		log.Error(err)
		return nil, false
	}

	sentTimestamp := content.SignedForeignLatency.Timestamp

	//check freshness
	if !isFresh(sentTimestamp, freshnessDelta) {
		log.Error("Not fresh enough")
		return nil, false
	}

	//Check message 5 sent after message 4
	if sentTimestamp.Before(latencyConstr.LocalTimestamps[1]) {
		log.Error("Too old timestamp")
		return nil, false
	}

	newLatency := &ConfirmedLatency{
		Latency:            latencyConstr.Latency,
		Timestamp:          sentTimestamp,
		SignedConfirmation: content.DoubleSignedForeignLatency,
	}

	log.LLvl1("Returning new latency from message 5")

	return newLatency, true

}

func isFresh(timestamp time.Time, delta time.Duration) bool {
	return timestamp.After(time.Now().Add(-delta))
}

func acceptableDifference(time1 time.Duration, time2 time.Duration, delta time.Duration) bool {
	return time1-time2 < delta && time2-time1 < delta
}
