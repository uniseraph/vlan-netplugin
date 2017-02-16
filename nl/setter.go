package nl

import (
	"net"

	"github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

type networkSetter func(netlink.Link) error

func MacSetter(mac net.HardwareAddr) networkSetter {
	return func(link netlink.Link) error {
		logrus.WithFields(
			logrus.Fields{"name": link.Attrs().Name, "mac": mac},
		).Debug("set eth mac address")
		return netlink.LinkSetHardwareAddr(link, mac)
	}
}

func UpSetter() networkSetter {
	return func(link netlink.Link) error {
		if link.Attrs().Flags&net.FlagUp == 1 {
			return nil
		}
		logrus.WithFields(logrus.Fields{"name": link.Attrs().Name}).Debug("set eth up")
		return netlink.LinkSetUp(link)
	}
}

func JoinNetworkSetter(b *netlink.Bridge) networkSetter {
	return func(link netlink.Link) error {
		if link.Attrs().MasterIndex == b.Index {
			return nil
		}
		logrus.WithFields(
			logrus.Fields{"name": link.Attrs().Name, "bridge": b.Name},
		).Debug("bridge add interface")
		return netlink.LinkSetMaster(link, b)
	}
}

func Set(link netlink.Link, setters ...networkSetter) error {
	for _, setter := range setters {
		if err := setter(link); err != nil {
			return err
		}
	}
	return nil
}
