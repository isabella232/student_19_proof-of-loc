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
func InitListening(srcAddress string, finish <-chan bool, ready chan<- bool, wg *sync.WaitGroup) chan PingMsg {
	log.LLvl1("Init UDP listening on " + srcAddress)
	receive := make(chan PingMsg, 100)
	wg.Add(1)
	go listen(receive, srcAddress, finish, ready, wg)
	return receive
}

func listen(receive chan PingMsg, srcAddress string, finish <-chan bool, ready chan<- bool, wg *sync.WaitGroup) {

	log.LLvl1("Setting up address")
	nodeAddress, _ := net.ResolveUDPAddr("udp", srcAddress)

	log.LLvl1("Open connection")
	connection, err := net.ListenUDP("udp", nodeAddress)
	if err != nil {
		return
	}
	defer connection.Close()

	ready <- true
	for {
		select {
		case <-finish:
			log.LLvl1("Listen stopping")
			connection.Close()
			wg.Done()
			return
		default:
			inputBytes := make([]byte, 100000)
			connection.SetReadDeadline(time.Now().Add(5 * time.Millisecond))
			len, _, err := connection.ReadFrom(inputBytes)
			if err != nil && !strings.Contains(err.Error(), "i/o timeout") {
				log.LLvl1(err)
			}
			if len > 0 {
				log.LLvl1("Received message")
				var msg PingMsg
				err = protobuf.Decode(inputBytes, &msg)
				if err != nil {
					log.LLvl1(err)
				}
				log.LLvl1("Received message from " + msg.Src.Address.String())
				receive <- msg
			}
		}
	}
}

//SendMessage lets a server ping another server
func SendMessage(message PingMsg, srcAddress string, dstAddress string) error {
	log.LLvl1("Sending message to " + dstAddress)
	destinationAddress, _ := net.ResolveUDPAddr("udp", dstAddress)
	sourceAddress, _ := net.ResolveUDPAddr("udp", srcAddress)

	connection, err := net.DialUDP("udp", sourceAddress, destinationAddress)
	if err != nil {
		log.LLvl1("Could not dial up")
		return err
	}
	defer connection.Close()

	encoded, err := protobuf.Encode(&message)
	if err != nil {
		log.LLvl1("Could not encode message")
		return err
	}
	log.LLvl1("Writing message to channel")
	connection.Write(encoded)
	return nil
}
