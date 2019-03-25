package latencyprotocol

// sources: https://holwech.github.io/blog/Creating-a-simple-UDP-module/
import (
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
	"net"
)

//InitListening allows the start of listening for pings on the server
func InitListening(srcAddress string, finish <-chan bool) <-chan PingMsg {
	log.LLvl1("Init UDP listening on " + srcAddress)
	receive := make(chan PingMsg, 10)
	go listen(receive, srcAddress, finish)
	return receive
}

func listen(receive chan PingMsg, srcAddress string, finish <-chan bool) {

	log.LLvl1("Setting up address")
	nodeAddress, _ := net.ResolveUDPAddr("udp", srcAddress)

	log.LLvl1("Open connection")
	connection, err := net.ListenUDP("udp", nodeAddress)
	if err != nil {
		return
	}
	defer connection.Close()

	log.LLvl1("Wait for message")
	var msg PingMsg
	for {
		select {
		case <-finish:
			return
		default:
			inputBytes := make([]byte, 4096)
			err := protobuf.Decode(inputBytes, &msg)
			if err != nil {
				log.LLvl1(err)
			}
			receive <- msg
		}
	}
}

//SendMessage lets a server ping another server
func SendMessage(message PingMsg, srcAddress string, dstAddress string) error {
	destinationAddress, _ := net.ResolveUDPAddr("udp", dstAddress)
	sourceAddress, _ := net.ResolveUDPAddr("udp", srcAddress)

	connection, err := net.DialUDP("udp", sourceAddress, destinationAddress)
	if err != nil {
		return err
	}
	defer connection.Close()

	encoded, err := protobuf.Encode(&message)
	if err != nil {
		return err
	}
	connection.Write(encoded)
	return nil
}

/*c, err := icmp.ListenPacket("udp6", "fe80::1%en0")
if err != nil {
    log.Fatal(err)
}
defer c.Close()

wm := icmp.Message{
    Type: ipv6.ICMPTypeEchoRequest, Code: 0,
    Body: &icmp.Echo{
        ID: os.Getpid() & 0xffff, Seq: 1,
        Data: []byte("HELLO-R-U-THERE"),
    },
}
wb, err := wm.Marshal(nil)
if err != nil {
    log.Fatal(err)
}
if _, err := c.WriteTo(wb, &net.UDPAddr{IP: net.ParseIP("ff02::1"), Zone: "en0"}); err != nil {
    log.Fatal(err)
}

rb := make([]byte, 1500)
n, peer, err := c.ReadFrom(rb)
if err != nil {
    log.Fatal(err)
}
rm, err := icmp.ParseMessage(58, rb[:n])
if err != nil {
    log.Fatal(err)
}
switch rm.Type {
case ipv6.ICMPTypeEchoReply:
    log.Printf("got reflection from %v", peer)
default:
    log.Printf("got %+v; want echo reply", rm)
}*/
