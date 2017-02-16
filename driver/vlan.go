package driver

import (
	"sync"
	"github.com/docker/libkv/store"
	"github.com/omega/vlan-netplugin/nl"
	"github.com/docker/go-plugins-helpers/network"
	"errors"
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
	return  &network.CapabilitiesResponse{
		Scope: network.GlobalScope,
	} ,nil
}

func (d *Driver) CreateNetwork(r *network.CreateNetworkRequest) error {
	n := &Network{r}

	_ , err := n.VlanId()
	if err!=nil {
		return err
	}

	return d.networks.Put(n)

}

func (*Driver) AllocateNetwork(*network.AllocateNetworkRequest) (*network.AllocateNetworkResponse, error) {
	panic("implement me")
}

func (d *Driver) DeleteNetwork( r *network.DeleteNetworkRequest) error {
	if _, err := d.networks.Get(r.NetworkID); err != nil {
		if err == store.ErrKeyNotFound {
			return nil
		}
		return err
	}
	return d.networks.Delete(r.NetworkID)}

func (*Driver) FreeNetwork(*network.FreeNetworkRequest) error {
return nil
}

func (d *Driver) CreateEndpoint(r *network.CreateEndpointRequest) (*network.CreateEndpointResponse, error) {
	var resp network.CreateEndpointResponse

	if r.Interface.Address == "" {
		return nil, errors.New("CreateEndpointRequest.Interface.Address must be specified")
	}

	ep := &Endpoint{r}

	if ep.Interface.MacAddress == "" {
		ep.GenerateMacAddress()
		resp.Interface = &network.EndpointInterface{MacAddress: ep.Interface.MacAddress}
	}

	if err := d.endpoints.Put(ep); err != nil {
		return nil, err
	}

	return &resp, nil

}

func (d *Driver) DeleteEndpoint(r *network.DeleteEndpointRequest) error {

	_ , err := d.endpoints.Get(r.EndpointID)
	if err!=nil{
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

func (*Driver) Join(*network.JoinRequest) (*network.JoinResponse, error) {
	panic("implement me")
}

func (*Driver) Leave(*network.LeaveRequest) error {
	panic("implement me")
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

