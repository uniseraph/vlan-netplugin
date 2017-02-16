package nl

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func FindParentFromVlan(dev string) (string, error) {
	link, err := netlink.LinkByName(dev)
	if err != nil {
		return "", err
	}

	vlanDev, ok := link.(*netlink.Vlan)
	if !ok {
		return dev, nil
	}
	plink, err := netlink.LinkByIndex(vlanDev.ParentIndex)
	if err != nil {
		return "", err
	}
	return plink.Attrs().Name, nil
}

func PreferredParentEth() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	hosts, err := net.LookupHost(hostname)
	if err != nil {
		return "", err
	}
	if len(hosts) == 0 {
		return "", fmt.Errorf("cannot locate related address for host %q", hostname)
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, _, _ := net.ParseCIDR(addr.String())
			if ip.To4().String() == hosts[0] {
				return iface.Name, nil
			}
		}
	}

	return "", fmt.Errorf("cannot find preferred ethernet")
}

func CreateVethPeer(name string) ([]*netlink.Veth, error) {
	if _, err := netlink.LinkByName(name); err != nil {
		if err := netlink.LinkAdd(
			&netlink.Veth{LinkAttrs: netlink.LinkAttrs{Name: name}, PeerName: "v" + name},
		); err != nil {
			return nil, err
		}
		logrus.WithField("veth", name).Info("successfully create veth device")
	}

	origin, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}
	local, err := netlink.LinkByName("v" + name)
	if err != nil {
		return nil, err
	}

	return []*netlink.Veth{origin.(*netlink.Veth), local.(*netlink.Veth)}, nil
}

func DelVethPeer(name string) error {
	if link, err := netlink.LinkByName(name); err == nil {
		netlink.LinkSetDown(link)
		netlink.LinkDel(link)
	}

	if link, err := netlink.LinkByName("v" + name); err == nil {
		netlink.LinkSetDown(link)
		netlink.LinkDel(link)
	}
	return nil
}

func CreateVlan(parentName string, vlanId int, vlanName string) (*netlink.Vlan, error) {
	parent, err := netlink.LinkByName(parentName)
	if err != nil {
		return nil, err
	}

	if link, err := netlink.LinkByName(vlanName); err == nil {
		if vlan, ok := link.(*netlink.Vlan); ok {
			if vlan.ParentIndex == parent.Attrs().Index && vlan.VlanId == vlanId {
				logrus.WithField("vlan", vlanName).Debug("find exist vlan device")
				return vlan, nil // VLAN设备已经存在
			}
		}
		return nil, errors.New("another interface exists, but cannot be used")
	}

	if err = netlink.LinkAdd(&netlink.Vlan{
		LinkAttrs: netlink.LinkAttrs{Name: vlanName, ParentIndex: parent.Attrs().Index},
		VlanId:    vlanId,
	}); err != nil {
		return nil, err
	}
	logrus.WithField("vlan", vlanName).Info("successfully create vlan device")

	vlan, err := netlink.LinkByName(vlanName)
	if err != nil {
		return nil, err
	}
	return vlan.(*netlink.Vlan), nil
}

func CreateBridge(bridgeName string) (*netlink.Bridge, error) {
	if iface, err := netlink.LinkByName(bridgeName); err == nil {
		if bridge, ok := iface.(*netlink.Bridge); ok {
			logrus.WithField("bridge", bridgeName).Debug("find exist bridge device")
			return bridge, nil
		}
		return nil, errors.New("another interface exists, but not bridge")
	}

	if err := netlink.LinkAdd(&netlink.Bridge{netlink.LinkAttrs{Name: bridgeName}}); err != nil {
		return nil, err
	}
	logrus.WithField("bridge", bridgeName).Info("successfully create bridge device")

	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return nil, err
	}
	return bridge.(*netlink.Bridge), nil
}

func GetDevicesAttachedOnBridge(bridgeName string) ([]string, error) {
	iface, err := netlink.LinkByName(bridgeName)
	if err != nil {
		logrus.WithField("bridge", bridgeName).Error("bridge not found")
		return nil, err
	}

	bridge, ok := iface.(*netlink.Bridge)
	if !ok {
		logrus.WithField("bridge", bridgeName).Debug("device is not a bridge")
		return nil, errors.New("device is not a bridge")
	}

	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	devs := []string{}
	for _, link := range links {
		if link.Attrs().MasterIndex == bridge.Index {
			devs = append(devs, link.Attrs().Name)
		}
	}
	return devs, nil
}

func DestroyDevice(name string) error {
	if link, err := netlink.LinkByName(name); err == nil {
		netlink.LinkSetDown(link)
		return netlink.LinkDel(link)
	}
	return nil
}
