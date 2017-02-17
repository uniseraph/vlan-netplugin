package driver

import (
	"errors"
	"fmt"
	"github.com/docker/go-plugins-helpers/network"
	"github.com/docker/libkv/store"
	"github.com/docker/libnetwork/netlabel"
	"github.com/omega/vlan-netplugin/nl"
	"github.com/vishvananda/netlink"
	"net"
	"sync"
	"github.com/contiv/ofnet/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

const (
	driverType = "vlan"
)

type DriverOption struct {
	Store     store.Store
	Prefix    string
	ParentEth string
}

func (o DriverOption) Eth() (dev string, err error) {
	dev = o.ParentEth
	if dev == "" {
		if dev, err = nl.PreferredParentEth(); err != nil {
			return "", err
		}
	}
	return nl.FindParentFromVlan(dev)
}

func New(option DriverOption) (*Driver, error) {
	dev, err := option.Eth()
	if err != nil {
		return nil, err
	}
	setDefaultRootChains(option.Prefix)
	return &Driver{
		dev:       dev,
		networks:  Networks{option.Store},
		endpoints: Endpoints{option.Store},
	}, nil
}

type Driver struct {
	dev string

	networks  Networks
	endpoints Endpoints

	sync.Mutex
}

func (*Driver) GetCapabilities() (*network.CapabilitiesResponse, error) {
	return &network.CapabilitiesResponse{
		Scope: network.GlobalScope,
	}, nil
}

func (d *Driver) CreateNetwork(r *network.CreateNetworkRequest) error {
	n := &Network{r}

	_, err := n.VlanId()
	if err != nil {
		return err
	}

	return d.networks.Put(n)

}

func (*Driver) AllocateNetwork(*network.AllocateNetworkRequest) (*network.AllocateNetworkResponse, error) {
	return &network.AllocateNetworkResponse{}, nil
}

func (d *Driver) DeleteNetwork(r *network.DeleteNetworkRequest) error {
	if _, err := d.networks.Get(r.NetworkID); err != nil {
		if err == store.ErrKeyNotFound {
			return nil
		}
		return err
	}
	return d.networks.Delete(r.NetworkID)
}

func (*Driver) FreeNetwork(*network.FreeNetworkRequest) error {
	return nil
}

func (d *Driver) CreateEndpoint(r *network.CreateEndpointRequest) (*network.CreateEndpointResponse, error) {

	logrus.Info("Create a endpoint ")

	if r.Interface.Address == "" {
		return nil, errors.New("CreateEndpointRequest.Interface.Address must be specified")
	}

	if v, exists := r.Options[netlabel.PortMap]; exists {
		if pb, ok := v.([]interface{}); ok && len(pb) > 0 {
			return nil, errors.New(`NetworkDriver "vlan" doesn't support port mapping`)
		}
	}

	ep := &Endpoint{r}
	if ep.Interface.MacAddress == "" {
		ep.GenerateMacAddress()
	}

	if err := d.endpoints.Put(ep); err != nil {
		return nil, err
	}

	return &network.CreateEndpointResponse{
		Interface: &network.EndpointInterface{
			Address: ep.Interface.Address,
			AddressIPv6: ep.Interface.AddressIPv6,
			MacAddress: ep.Interface.MacAddress,
		} ,
	} ,nil

}

func (d *Driver) DeleteEndpoint(r *network.DeleteEndpointRequest) error {

	_, err := d.endpoints.Get(r.EndpointID)
	if err != nil {
		return err
	}

	return d.endpoints.Delete(r.EndpointID)
}

func (d *Driver) EndpointInfo(r *network.InfoRequest) (*network.InfoResponse, error) {
	var resp network.InfoResponse

	ep, err := d.endpoints.Get(r.EndpointID)
	if err != nil {
		return nil, err
	}

	resp.Value = make(map[string]string)
	if len(ep.Options) > 0 {
		for k, v := range ep.Options {
			if vs, ok := v.(string); ok {
				resp.Value[k] = vs
			}
		}
	}
	return &resp, nil
}

func (d *Driver) Join(r *network.JoinRequest) (*network.JoinResponse, error) {
	var err error

	n, err := d.networks.Get(r.NetworkID)
	if err != nil {
		return nil, err
	}
	vlanId, err := n.VlanId()
	if err != nil {
		return nil, err
	}

	vlanName := func() string {
		return fmt.Sprintf("%s.%d", d.dev, vlanId)
	}
	bridgeName := func() string {
		return fmt.Sprintf("br0.%d", vlanId)
	}

	ep, err := d.endpoints.Get(r.EndpointID)
	if err != nil {
		return nil, err
	}
	ipv4data, err := n.FindIPv4Data(ep.Interface.Address)
	if err != nil {
		return nil, err
	}
	gateway, _, err := net.ParseCIDR(ipv4data.Gateway)
	if err != nil {
		return nil, err
	}

	d.Lock()
	defer d.Unlock()

	vlanDev, err := nl.CreateVlan(d.dev, vlanId, vlanName())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			nl.DestroyDevice(vlanName())
		}
	}()

	bridgeDev, err := nl.CreateBridge(bridgeName())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			nl.DestroyDevice(bridgeName())
		}
	}()

	linkSetUp := nl.UpSetter()
	for _, link := range []netlink.Link{bridgeDev, vlanDev} {
		if err = linkSetUp(link); err != nil {
			return nil, err
		}
	}

	linkSetMaster := nl.JoinNetworkSetter(bridgeDev)
	if err = linkSetMaster(vlanDev); err != nil {
		return nil, err
	}

	veths, err := nl.CreateVethPeer(ep.VethName())
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			nl.DestroyDevice(veths[0].Attrs().Name)
		}
	}()

//	origin := veths[0] //peer in host , vxxxx
//	local := veths[1]  //peer in container , vvxxx

	if err = nl.Set(
		veths[0],
		nl.MacSetter(ep.VethSourceMacAddress()),
		nl.JoinNetworkSetter(bridgeDev),
		nl.UpSetter(),
	); err != nil {
		return nil, err
	}

	if err = nl.Set(veths[1], nl.MacSetter(ep.VethDstMacAddress())); err != nil {
		return nil, err
	}

	return &network.JoinResponse{
		InterfaceName: network.InterfaceName{veths[1].Attrs().Name, "eth"},
		Gateway:       gateway.String(),
	}, nil
}

func (d *Driver) Leave(r *network.LeaveRequest) error {
	ep, err := d.endpoints.Get(r.EndpointID)
	if err != nil {
		return err
	}

	d.Lock()
	defer d.Unlock()

	return nl.DestroyDevice(ep.VethName())
}

func (*Driver) DiscoverNew(*network.DiscoveryNotification) error {
	return nil
}

func (*Driver) DiscoverDelete(*network.DiscoveryNotification) error {
	return nil
}

func (*Driver) ProgramExternalConnectivity(*network.ProgramExternalConnectivityRequest) error {
	return nil
}

func (*Driver) RevokeExternalConnectivity(*network.RevokeExternalConnectivityRequest) error {
	return nil
}

func (d *Driver) Type() string {
	return driverType
}
