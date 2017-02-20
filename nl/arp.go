package nl

import (
	"bytes"
	"encoding/binary"
	"net"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

const (
	ARP_REQUEST = 1
	ARP_REPLY   = 2

	ETH_8021Q_TPID = 0x8100
)

var (
	ethAddrBroadcast   net.HardwareAddr
	ethAddrUnspecified net.HardwareAddr
)

func init() {
	var err error

	if ethAddrBroadcast, err = net.ParseMAC("FF:FF:FF:FF:FF:FF"); err != nil {
		panic(err)
	}
	if ethAddrUnspecified, err = net.ParseMAC("00:00:00:00:00:00"); err != nil {
		panic(err)
	}
}

func htons(n uint16) uint16 {
	var high uint16 = n >> 8
	return n<<8 + high
}

func createSocket() (int, error) {
	return syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_ARP)))
}

func getAddress(dev string) (syscall.Sockaddr, error) {
	link, err := netlink.LinkByName(dev)
	if err != nil {
		return nil, err
	}
	return &syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_ARP),
		Ifindex:  link.Attrs().Index,
	}, nil
}

type arpPacket struct {
	DestHwAddr [6]byte
	SrcHwAddr  [6]byte

	// 802.1Q header
	TPID uint16
	VID  uint16

	FrameType     uint16
	HwType        uint16
	ProtoType     uint16
	HwAddrSize    byte
	ProtoAddrSize byte
	Op            uint16
	SndrHwAddr    [6]byte
	SndrIpAddr    [4]byte
	RcptHwAddr    [6]byte
	RcptIpAddr    [4]byte
	padding       [18]byte
}

func createArpPacket(op uint16, srcMac net.HardwareAddr, srcIP, dstIP net.IP, vlanId int) *arpPacket {
	packet := &arpPacket{
		TPID:          htons(ETH_8021Q_TPID),
		VID:           htons(uint16(vlanId)),
		FrameType:     htons(syscall.ETH_P_ARP),
		HwType:        htons(syscall.ARPHRD_ETHER),
		ProtoType:     htons(syscall.ETH_P_IP),
		HwAddrSize:    6,
		ProtoAddrSize: 4,
		Op:            htons(op),
	}
	copy(packet.DestHwAddr[:], ethAddrBroadcast)
	copy(packet.SrcHwAddr[:], srcMac)
	copy(packet.SndrHwAddr[:], srcMac)
	copy(packet.RcptHwAddr[:], ethAddrUnspecified)
	copy(packet.SndrIpAddr[:], srcIP.To4()) // 这里必须To4否则可能为兼容IPv6的格式，高位填充0导致copy内容出错
	copy(packet.RcptIpAddr[:], dstIP.To4())
	return packet
}

func buildArpPacketBuffer(packet *arpPacket) ([]byte, error) {
	buffer := &bytes.Buffer{}
	if err := binary.Write(buffer, binary.LittleEndian, packet); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func SendArpRequest(srcMac net.HardwareAddr, srcIP, dstIP net.IP, dev string, vlanId int) error {
	socket, err := createSocket()
	if err != nil {
		logrus.Error("failed to create raw arp socket: ", err)
		return err
	}
	defer syscall.Close(socket)

	sockAddr, err := getAddress(dev)
	if err != nil {
		logrus.Error("failed to bind arp socket address: ", err)
		return err
	}

	buf, err := buildArpPacketBuffer(createArpPacket(ARP_REQUEST, srcMac, srcIP, dstIP, vlanId))
	if err != nil {
		logrus.Error("faeild to create arp packet: ", err)
		return err
	}

	logrus.WithField("src", srcIP).WithField("dst", dstIP).Debug("send arp request")
	return syscall.Sendto(socket, buf, 0, sockAddr)
}
