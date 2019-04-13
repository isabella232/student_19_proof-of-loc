package latencyprotocol

import (
	"errors"
	"github.com/dedis/student_19_proof-of-loc/knowthyneighbor/udp"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
	sigAlg "golang.org/x/crypto/ed25519"
	"math"
	"math/rand"
	"sync"
	"time"
)

const freshnessDelta = 10 * time.Second
const intervallDelta = 10 * time.Second

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

	nonce := Nonce(rand.Intn(math.MaxInt32))

	encodedKey := string(dstNodeID.PublicKey)
	_, alreadyStarted := Node.LatenciesInConstruction[encodedKey]

	if alreadyStarted {
		log.Warn("Already started messaging this node")
		return errors.New("Already started messaging this node")
	}

	srcAddress := Node.SendingAddress.NetworkAddress()
	dstAddress := dstNodeID.ServerID.Address.NetworkAddress()

	var wg sync.WaitGroup

	finishedSendingChan, MsgChan := udp.InitSending(srcAddress, dstAddress, &wg)

	timestamp := time.Now()

	msgContent := &PingMsg1{
		SrcNonce:  nonce,
		Timestamp: timestamp,
	}

	unsigned, err := protobuf.Encode(msgContent)
	if err != nil {
		log.Warn(err)
		return err
	}

	signed := sigAlg.Sign(Node.PrivateKey, unsigned)

	msg := udp.PingMsg{
		Src:       *Node.ID.ServerID,
		Dst:       *dstNodeID.ServerID,
		SeqNb:     1,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsigned,
		SignedContent:   signed,
	}

	MsgChan <- msg

	latConstr := LatencyConstructor{
		StartedLocally:    true,
		CurrentMsgNb:      1,
		DstID:             dstNodeID,
		Nonce:             nonce,
		LocalTimestamps:   make([]time.Time, 2),
		ForeignTimestamps: make([]time.Time, 2),
		ClockSkews:        make([]time.Duration, 2),
		Latency:           0,
		SignedLatency:     nil,
		MsgChannel:        &MsgChan,
		FinishedSending:   &finishedSendingChan,
		WaitGroup:         &wg,
	}

	latConstr.LocalTimestamps[0] = timestamp

	Node.LatenciesInConstruction[encodedKey] = &latConstr

	return nil

}

func (Node *Node) checkMessage1(msg *udp.PingMsg) (*PingMsg1, bool) {
	newPubKey := msg.PublicKey

	sigCorrect := sigAlg.Verify(newPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		log.Warn("Incorrect signature from message")
		return nil, false
	}

	encodedKey := string(newPubKey)
	_, alreadyStarted := Node.LatenciesInConstruction[encodedKey]

	if alreadyStarted {
		log.Warn("Already started messaging this node")
		return nil, false
	}

	content := PingMsg1{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		log.Warn("Could not decode message")
		return nil, false
	}

	if !isFresh(content.Timestamp, freshnessDelta) {
		log.Warn("Timestamp too old")
		return nil, false
	}

	return &content, true

}

func (Node *Node) sendMessage2(msg *udp.PingMsg, msgContent *PingMsg1) error {

	srcAddress := Node.SendingAddress.NetworkAddress()
	dstAddress := msg.Src.Address.NetworkAddress()

	var wg sync.WaitGroup

	finishedSendingChan, MsgChan := udp.InitSending(srcAddress, dstAddress, &wg)

	nonce := Nonce(rand.Intn(math.MaxInt32))

	localtime := time.Now()

	msg2Content := &PingMsg2{
		SrcNonce:  nonce,
		DstNonce:  msgContent.SrcNonce,
		Timestamp: localtime,
	}

	unsigned, err := protobuf.Encode(msg2Content)
	if err != nil {
		log.Warn(err)
		return err
	}

	signed := sigAlg.Sign(Node.PrivateKey, unsigned)

	newMsg := udp.PingMsg{
		Src:       msg.Dst,
		Dst:       msg.Src,
		SeqNb:     2,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsigned,
		SignedContent:   signed,
	}

	MsgChan <- newMsg

	latencyConstr := LatencyConstructor{
		StartedLocally:    false,
		CurrentMsgNb:      2,
		DstID:             &NodeID{&msg.Src, msg.PublicKey},
		Nonce:             nonce,
		LocalTimestamps:   make([]time.Time, 2),
		ForeignTimestamps: make([]time.Time, 2),
		ClockSkews:        make([]time.Duration, 2),
		Latency:           0,
		SignedLatency:     nil,
		MsgChannel:        &MsgChan,
		FinishedSending:   &finishedSendingChan,
		WaitGroup:         &wg,
	}

	encodedKey := string(msg.PublicKey)
	Node.LatenciesInConstruction[encodedKey] = &latencyConstr

	latencyConstr.LocalTimestamps[0] = localtime
	latencyConstr.ForeignTimestamps[0] = msgContent.Timestamp
	latencyConstr.ClockSkews[0] = localtime.Sub(msgContent.Timestamp)

	return nil

}

func (Node *Node) checkMessage2(msg *udp.PingMsg) (*PingMsg2, bool) {

	senderPubKey := msg.PublicKey

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		log.Warn("Signature incorrect")
		return nil, false
	}

	//extract content
	content := PingMsg2{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		log.Warn(err)
		return nil, false
	}

	//check freshness
	if !isFresh(content.Timestamp, freshnessDelta) {
		log.Warn("Not fresh enough")
		return nil, false
	}

	//check if we are building a latency for this
	encodedKey := string(senderPubKey)
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[encodedKey]

	if !alreadyStarted {
		log.Warn("Not started yet")
		return nil, false
	}

	//Check message 2 sent after message 1
	if content.Timestamp.Before(latencyConstr.LocalTimestamps[0]) {
		log.Warn("Timestamp order wrong")
		return nil, false
	}

	//check nonce
	if content.DstNonce != latencyConstr.Nonce {
		log.Warn("Wrong nonce")
		return nil, false
	}

	return &content, true

}

func (Node *Node) sendMessage3(msg *udp.PingMsg, msgContent *PingMsg2) error {

	encodedKey := string(msg.PublicKey)
	latencyConstr := Node.LatenciesInConstruction[encodedKey]

	localtime := time.Now()

	latency := localtime.Sub(latencyConstr.LocalTimestamps[0])

	unsignedLatency, err := protobuf.Encode(&LatencyWrapper{latency})
	if err != nil {
		log.Warn(err)
		return err
	}

	signedLatency := sigAlg.Sign(Node.PrivateKey, unsignedLatency)

	msg3Content := &PingMsg3{
		DstNonce:      msgContent.SrcNonce,
		Latency:       latency,
		SignedLatency: signedLatency,
	}

	unsignedContent, err := protobuf.Encode(msg3Content)
	if err != nil {
		log.Warn(err)
		return err
	}

	signedContent := sigAlg.Sign(Node.PrivateKey, unsignedContent)

	newMsg := udp.PingMsg{
		Src:             msg.Dst,
		Dst:             msg.Src,
		SeqNb:           3,
		PublicKey:       Node.ID.PublicKey,
		UnsignedContent: unsignedContent,
		SignedContent:   signedContent,
	}

	*latencyConstr.MsgChannel <- newMsg

	latencyConstr.CurrentMsgNb += 2
	latencyConstr.LocalTimestamps[1] = localtime
	latencyConstr.ForeignTimestamps[0] = msgContent.Timestamp
	latencyConstr.ClockSkews[0] = localtime.Sub(msgContent.Timestamp)
	latencyConstr.Latency = latency
	latencyConstr.SignedLatency = signedLatency

	return nil

}

func (Node *Node) checkMessage3(msg *udp.PingMsg) (*PingMsg3, bool) {

	senderPubKey := msg.PublicKey

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		log.Warn("Signature incorrect")
		return nil, false
	}

	//extract content
	content := PingMsg3{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		log.Warn(err)
		return nil, false
	}

	//check if we are building a latency for this
	encodedKey := string(senderPubKey)
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[encodedKey]

	if !alreadyStarted {
		log.Warn("Not started yet")
		return nil, false
	}

	sigTimestamp := latencyConstr.ForeignTimestamps[0].Add(content.Latency)
	latencyConstr.ForeignTimestamps[1] = sigTimestamp

	//check freshness
	if !isFresh(sigTimestamp, freshnessDelta) {
		log.Warn("Old message")
		return nil, false
	}

	//check nonce
	if content.DstNonce != latencyConstr.Nonce {
		log.Warn("Nonce wrong")
		return nil, false
	}

	return &content, true

}

func (Node *Node) sendMessage4(msg *udp.PingMsg, msgContent *PingMsg3) error {

	encodedKey := string(msg.PublicKey)
	latencyConstr := Node.LatenciesInConstruction[encodedKey]

	localtime := time.Now()
	latencyConstr.ClockSkews[1] = localtime.Sub(latencyConstr.ForeignTimestamps[1])

	if !acceptableDifference(latencyConstr.ClockSkews[0], latencyConstr.ClockSkews[1], intervallDelta) {
		log.Warn("Clock Skews too different")
		return errors.New("Clock Skews too different")
	}

	localLatency := localtime.Sub(latencyConstr.LocalTimestamps[0])

	if !acceptableDifference(localLatency, msgContent.Latency, intervallDelta) {
		log.Warn("Latencies too different")
		return errors.New("Latencies too different")
	}

	unsignedLocalLatency, err := protobuf.Encode(&LatencyWrapper{localLatency})
	if err != nil {
		log.Warn(err)
		return err
	}
	signedLocalLatency := sigAlg.Sign(Node.PrivateKey, unsignedLocalLatency)

	signedForeignLatency := SignedForeignLatency{localtime, msgContent.SignedLatency}
	signedForeignLatencyBytes, err := protobuf.Encode(&signedForeignLatency)
	if err != nil {
		log.Warn(err)
		return err
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
		log.Warn(err)
		return err
	}

	signedContent := sigAlg.Sign(Node.PrivateKey, unsignedContent)

	newMsg := udp.PingMsg{
		Src:       msg.Dst,
		Dst:       msg.Src,
		SeqNb:     4,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsignedContent,
		SignedContent:   signedContent,
	}

	*latencyConstr.MsgChannel <- newMsg
	*latencyConstr.FinishedSending <- true

	latencyConstr.Latency = localLatency
	latencyConstr.LocalTimestamps[1] = localtime
	latencyConstr.SignedLatency = signedLocalLatency
	latencyConstr.CurrentMsgNb += 2

	latencyConstr.WaitGroup.Wait()

	return nil

}

func (Node *Node) checkMessage4(msg *udp.PingMsg) (*PingMsg4, bool) {

	senderPubKey := msg.PublicKey

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		log.Warn("Signature incorrect")
		return nil, false
	}

	//extract content
	content := PingMsg4{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		log.Warn(err)
		return nil, false
	}

	//check if we are building a latency for this
	encodedKey := string(msg.PublicKey)
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[encodedKey]

	if !alreadyStarted {
		log.Warn("Not started yet")
		return nil, false
	}

	sentTimestamp := content.SignedForeignLatency.Timestamp
	latencyConstr.ForeignTimestamps[1] = sentTimestamp

	//check freshness
	if !isFresh(sentTimestamp, freshnessDelta) {
		log.Warn("Not fresh enough")
		return nil, false
	}

	return &content, true

}

func (Node *Node) sendMessage5(msg *udp.PingMsg, msgContent *PingMsg4) (*ConfirmedLatency, error) {

	encodedKey := string(msg.PublicKey)
	latencyConstr := Node.LatenciesInConstruction[encodedKey]

	localtime := time.Now()
	latencyConstr.ClockSkews[1] = localtime.Sub(latencyConstr.ForeignTimestamps[1])

	if !acceptableDifference(latencyConstr.ClockSkews[0], latencyConstr.ClockSkews[1], intervallDelta) {
		log.Warn("Clock Skews too different")
		return nil, errors.New("Clock Skews too different")
	}

	if !acceptableDifference(latencyConstr.Latency, msgContent.LocalLatency, intervallDelta) {
		log.Warn("Latencies too different")
		return nil, errors.New("Latencies too different")
	}

	signedForeignLatency := SignedForeignLatency{localtime, msgContent.SignedLocalLatency}
	signedForeignLatencyBytes, err := protobuf.Encode(&signedForeignLatency)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	doubleSignedforeignLatency := sigAlg.Sign(Node.PrivateKey, signedForeignLatencyBytes)

	msg5Content := &PingMsg5{
		SignedForeignLatency:       signedForeignLatency,
		DoubleSignedForeignLatency: doubleSignedforeignLatency,
	}

	unsignedContent, err := protobuf.Encode(msg5Content)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	signedContent := sigAlg.Sign(Node.PrivateKey, unsignedContent)

	newMsg := udp.PingMsg{
		Src:       msg.Dst,
		Dst:       msg.Src,
		SeqNb:     5,
		PublicKey: Node.ID.PublicKey,

		UnsignedContent: unsignedContent,
		SignedContent:   signedContent,
	}

	*latencyConstr.MsgChannel <- newMsg
	*latencyConstr.FinishedSending <- true

	latencyConstr.CurrentMsgNb += 2
	latencyConstr.LocalTimestamps[1] = localtime

	newLatency := &ConfirmedLatency{
		Latency:            latencyConstr.Latency,
		SignedLatency:      latencyConstr.SignedLatency,
		Timestamp:          latencyConstr.ForeignTimestamps[1],
		SignedConfirmation: msgContent.DoubleSignedForeignLatency,
	}

	latencyConstr.WaitGroup.Wait()

	return newLatency, nil

}

func (Node *Node) checkMessage5(msg *udp.PingMsg) (*ConfirmedLatency, bool) {

	senderPubKey := msg.PublicKey

	//check signature
	sigCorrect := sigAlg.Verify(senderPubKey, msg.UnsignedContent, msg.SignedContent)
	if !sigCorrect {
		log.Warn("Signature incorrect")
		return nil, false
	}

	//extract content
	content := PingMsg5{}
	err := protobuf.Decode(msg.UnsignedContent, &content)
	if err != nil {
		log.Warn(err)
		return nil, false
	}

	sentTimestamp := content.SignedForeignLatency.Timestamp

	//check freshness
	if !isFresh(sentTimestamp, freshnessDelta) {
		log.Warn("Not fresh enough")
		return nil, false
	}

	//check if we are building a latency for this
	encodedKey := string(msg.PublicKey)
	latencyConstr, alreadyStarted := Node.LatenciesInConstruction[encodedKey]

	if !alreadyStarted {
		log.Warn("Already started")
		return nil, false
	}

	//Check message 5 sent after message 4
	if sentTimestamp.Before(latencyConstr.LocalTimestamps[1]) {
		log.Warn("Too old timestamp")
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
