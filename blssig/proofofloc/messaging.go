package proofofloc

// sources: https://holwech.github.io/blog/Creating-a-simple-UDP-module/
import (
	"bytes"
	"encoding/gob"
	"go.dedis.ch/onet/v3/network"
	"net"
)

func (Node *Node) InitMessaging(readPort string, writePort string, dstAddress network.Address) (<-chan PingMsg, chan<- PingMsg) {
	receive := make(chan PingMsg, 10)
	send := make(chan PingMsg, 10)
	go listen(receive, readPort)
	return receive, send
}

func listen(receive chan PingMsg, port string) {
	localAddress, _ := net.ResolveUDPAddr("udp", port)
	connection, err := net.ListenUDP("udp", localAddress)
	defer connection.Close()
	var message PingMsg
	for {
		inputBytes := make([]byte, 4096)
		length, _, _ := connection.ReadFromUDP(inputBytes)
		buffer := bytes.NewBuffer(inputBytes[:length])
		decoder := gob.NewDecoder(buffer)
		decoder.Decode(&message)
		receive <- message
	}
}

func SendMessage(message PingMsg, port string, srcAddress string, dstAddress string) {
	destinationAddress, _ := net.ResolveUDPAddr("udp", dstAddress)
	sourceAddress, _ := net.ResolveUDPAddr("udp", srcAddress)
	connection, err := net.DialUDP("udp", sourceAddress, destinationAddress)
	defer connection.Close()
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	for {
		encoder.Encode(message)
		connection.Write(buffer.Bytes())
		buffer.Reset()
	}
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
