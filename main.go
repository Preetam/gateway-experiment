package main

import (
	"github.com/PreetamJinka/proto"

	"flag"
	"log"
	"net"
	"syscall"
)

// host order (usually little endian) -> network order (big endian)
func htons(n int) int {
	return int(int16(byte(n))<<8 | int16(byte(n>>8)))
}

func main() {
	iface := flag.String("iface", "eth0", "interface to send packets to")
	bbbIface := flag.String("board-iface", "eth1", "interface that the board is connected to")
	flag.Parse()

	// Get the interface index
	netIface, err := net.InterfaceByName(*iface)
	if err != nil {
		log.Fatal(err)
	}

	boardIface, err := net.InterfaceByName(*bbbIface)
	if err != nil {
		log.Fatal(err)
	}

	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, htons(syscall.ETH_P_ALL))
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Listening on a raw socket...")

	log.Println(syscall.Sendto(fd, (protodecode.EthernetFrame{
		Source:      netIface.HardwareAddr,
		Destination: netIface.HardwareAddr,
		Payload:     []byte("hey wassup"),
	}).Bytes(), 0, &syscall.SockaddrLinklayer{Ifindex: netIface.Index}))

	buf := make([]byte, 65535)
	for {
		// "receive from" the socket into the buf
		n, _, err := syscall.Recvfrom(fd, buf, 0)
		if err != nil {
			log.Fatal(err)
		}

		ethernetPacket := protodecode.DecodeEthernet(buf[:n])
		if ethernetPacket.EtherType == 0x0800 {
			ipv4Packet := protodecode.DecodeIPv4(ethernetPacket.Payload)

			if ipv4Packet.Source.String() == "192.168.7.2" && ipv4Packet.Destination.String() != "192.168.7.1" {
				log.Println("this one needs to go to the gateway")

				log.Print(ethernetPacket)

				ethernetPacket.Source = net.HardwareAddr{0xe8, 0xde, 0x27, 0xbb, 0x6b, 0xaa}
				ethernetPacket.Destination = netIface.HardwareAddr

				log.Print(ethernetPacket)

				log.Println(syscall.Sendto(fd, ethernetPacket.Bytes(), 0, &syscall.SockaddrLinklayer{Ifindex: netIface.Index}))

				continue
			}

			if ipv4Packet.Destination.String() == "192.168.7.2" && ipv4Packet.Source.String() != "192.168.7.1" {
				log.Println("this one needs to go to the BeagleBone Black")

				ethernetPacket.Destination = netIface.HardwareAddr
				ethernetPacket.Source = net.HardwareAddr{0xbe, 0x1f, 0x62, 0xd8, 0x7d, 0x78}

				log.Println(syscall.Sendto(fd, ethernetPacket.Bytes(), 0, &syscall.SockaddrLinklayer{Ifindex: boardIface.Index}))

				continue
			}
		}
	}
}
