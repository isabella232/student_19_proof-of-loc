package udp

// sources: https://holwech.github.io/blog/Creating-a-simple-UDP-module/
import (
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
	sigAlg "golang.org/x/crypto/ed25519"
	"net"
	"strings"
	"sync"
	"time"
)

const checkForStopSignal = time.Duration(50 * time.Microsecond)
const readMessageSize = 1024

//PingMsg represents a message sent to another validator
type PingMsg struct {
	Src       network.ServerIdentity
	Dst       network.ServerIdentity
	SeqNb     int
	PublicKey sigAlg.PublicKey

	UnsignedContent []byte
	SignedContent   []byte
}

//InitListening allows the start of listening for pings on the server
func InitListening(srcAddress string, wg *sync.WaitGroup) (chan PingMsg, chan bool, error) {
	receive := make(chan PingMsg, 10)
	finish := make(chan bool, 1)

	nodeAddress, _ := net.ResolveUDPAddr("udp", srcAddress)

	connection, err := net.ListenUDP("udp", nodeAddress)
	if err != nil {
		return nil, nil, err
	}

	wg.Add(2)
	go stopListening(finish, connection, wg)
	go listen(receive, srcAddress, finish, connection, wg)
	return receive, finish, nil
}

func stopListening(finish chan bool, connection *net.UDPConn, wg *sync.WaitGroup) {
	<-finish
	connection.Close()
	wg.Done()
	return
}

func listen(receive chan PingMsg, srcAddress string, finish <-chan bool, connection *net.UDPConn, wg *sync.WaitGroup) {

	inputBytes := make([]byte, readMessageSize)
	for {
		len, _, err := connection.ReadFrom(inputBytes)
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				//indicates that the network was closed and we need to stop listening
				wg.Done()
				return
			}
			log.Warn(err)
		}
		if len > 0 {

			var msg PingMsg
			err = protobuf.Decode(inputBytes, &msg)
			/*if err != nil {
				log.Warn(err)
			}*/
			receive <- msg
			inputBytes = make([]byte, readMessageSize)
		}
	}
}

//InitSending allows the start of listening for pings on the server
func InitSending(srcAddress string, dstAddress string, wg *sync.WaitGroup) (chan bool, chan PingMsg) {
	msgChannel := make(chan PingMsg, 10)
	finish := make(chan bool, 1)
	wg.Add(1)
	go sendMessage(srcAddress, dstAddress, finish, msgChannel, wg)
	return finish, msgChannel
}

//sendMessage allows the start of sending between a source and a destination
func sendMessage(srcAddress string, dstAddress string, finish <-chan bool, msgChannel <-chan PingMsg, wg *sync.WaitGroup) error {

	destinationAddress, _ := net.ResolveUDPAddr("udp", dstAddress)
	sourceAddress, _ := net.ResolveUDPAddr("udp", srcAddress)

	connection, err := net.DialUDP("udp", sourceAddress, destinationAddress)
	if err != nil {
		log.Warn("Could not dial up")
		wg.Done()
		return err
	}
	for {
		select {
		case <-finish:
			connection.Close()
			wg.Done()
			return nil
		case message := <-msgChannel:
			encoded, err := protobuf.Encode(&message)
			if err != nil {
				log.Warn("Could not encode message")
				connection.Close()
				return err
			}
			connection.Write(encoded)
		}
	}
}
