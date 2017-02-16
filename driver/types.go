package driver

import (
	"errors"
	"fmt"
	"github.com/docker/go-plugins-helpers/network"
	"net"
	"strconv"
)

var (
	errVlanIdRequired  = errors.New(`opt "VlanId" must be specified`)
	errVlanIdIsInvalid = errors.New(`opt "VlanId" invalid, must 0 < VlanId < 4096`)
)

type Network struct {
	*network.CreateNetworkRequest
}
type Endpoint struct {
	*network.CreateEndpointRequest
}

func (n *Network) VlanId() (vlanId int, err error) {
	const netlabel = "com.docker.network.generic"

	if n.Options == nil {
		return 0, errVlanIdRequired
	}
	if _, exists := n.Options[netlabel]; !exists {
		return 0, errVlanIdRequired
	}

	genericOpt, ok := n.Options[netlabel].(map[string]interface{})
	if !ok {
		return 0, errVlanIdRequired
	}

	if v, exists := genericOpt["VlanId"]; exists {
		switch v.(type) {
		case string:
			vlanId, err = strconv.Atoi(v.(string))
		case float64:
			vlanId = int(v.(float64))
		case int:
			vlanId = v.(int)
		default:
			return 0, errVlanIdIsInvalid
		}

		if err != nil || vlanId <= 0 || vlanId >= 4096 {
			return 0, errVlanIdIsInvalid
		}
		return vlanId, nil
	}

	return 0, errVlanIdRequired
}

func (n *Network) FindIPv4Data(addr string) (*network.IPAMData, error) {
	ip, _, err := net.ParseCIDR(addr)
	if err != nil {
		return nil, err
	}

	for _, ipv4data := range n.IPv4Data {
		_, subnet, err := net.ParseCIDR(ipv4data.Pool)
		if err != nil {
			return nil, err
		}
		if subnet.Contains(ip) {
			return ipv4data, nil
		}
	}
	return nil, fmt.Errorf("cannot find matched subnet: ip=%s network=%s", addr, n.NetworkID)
}

func (e *Endpoint) GenerateMacAddress() {
	if e.Interface.MacAddress != "" {
		return
	}

	hw := make(net.HardwareAddr, 6)
	hw[0] = 0x7a
	hw[1] = 0x42
	copy(hw[2:], net.ParseIP(e.Interface.Address).To4())
	e.Interface.MacAddress = hw.String()
}

func (e *Endpoint) VethName() string {
	return "v" + e.EndpointID[:12]
}

func (e *Endpoint) VethSourceMacAddress() net.HardwareAddr {
	mac, _ := net.ParseMAC("FE:FF:FF:FF:FF:FF")
	return mac
}

func (e *Endpoint) VethDstMacAddress() net.HardwareAddr {
	mac, _ := net.ParseMAC(e.Interface.MacAddress)
	return mac
}
