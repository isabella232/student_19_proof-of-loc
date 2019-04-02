// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package latencyprotocol

import (
	"errors"
	"log"
	"net"
	"os"
	"runtime"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

//Ping uses ICMP to ping an IP address
func Ping(srcAddress string, dstAddress string) (time.Duration, error) {
	switch runtime.GOOS {
	case "darwin":
	case "linux":
		log.Println("you may need to adjust the net.ipv4.ping_group_range kernel state")
	default:
		log.Println("not supported on", runtime.GOOS)
		return 0, errors.New("not supported on " + runtime.GOOS)
	}

	c, err := icmp.ListenPacket("udp6", srcAddress)
	if err != nil {
		return 0, err
	}
	defer c.Close()

	wm := icmp.Message{
		Type: ipv6.ICMPTypeEchoRequest, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte("ping latency test"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return 0, err
	}

	sendingTime := time.Now()
	if _, err := c.WriteTo(wb, &net.UDPAddr{IP: net.ParseIP(dstAddress), Zone: "en0"}); err != nil {
		return 0, err
	}

	rb := make([]byte, 1500)
	n, peer, err := c.ReadFrom(rb)
	if err != nil {
		return 0, err
	}

	latency := time.Now().Sub(sendingTime)
	rm, err := icmp.ParseMessage(58, rb[:n])
	if err != nil {
		return 0, err
	}
	switch rm.Type {
	case ipv6.ICMPTypeEchoReply:
		log.Printf("got reflection from %v", peer)
		return latency, nil
	default:
		log.Printf("got %+v; want echo reply", rm)
	}

	return 0, errors.New("Did not receive reply")

}
