package driver

import (
	"encoding/json"
	"strings"

	"github.com/docker/libkv/store"
)

var (
	defaultRootChains = []string{"omega", "network", "vlan", "v1.0", driverType}
	rootChains        = []string{}
)

func setDefaultRootChains(prefix string) {
	for _, key := range strings.Split(prefix, "/") {
		if key != "" {
			rootChains = append(rootChains, key)
		}
	}
	rootChains = append(rootChains, defaultRootChains...)
}

func normalize(keys ...string) string {
	return strings.Join(append(rootChains, keys...), "/")
}

type Networks struct {
	s store.Store
}

func (ns Networks) Get(id string) (*Network, error) {
	kv, err := ns.s.Get(normalize("network", id))
	if err != nil {
		return nil, err
	}

	var network *Network
	if err := json.Unmarshal(kv.Value, &network); err != nil {
		return nil, err
	}
	return network, nil
}

func (ns Networks) Put(network *Network) error {
	data, err := json.Marshal(network)
	if err != nil {
		return err
	}
	return ns.s.Put(normalize("network", network.NetworkID), data, nil)
}

func (ns Networks) Delete(id string) error {
	return ns.s.Delete(normalize("network", id))
}

type Endpoints struct {
	s store.Store
}

func (es Endpoints) Get(id string) (*Endpoint, error) {
	kv, err := es.s.Get(normalize("endpoint", id))
	if err != nil {
		return nil, err
	}

	var endpoint *Endpoint
	if err := json.Unmarshal(kv.Value, &endpoint); err != nil {
		return nil, err
	}
	return endpoint, nil
}

func (es Endpoints) Put(endpoint *Endpoint) error {
	data, err := json.Marshal(endpoint)
	if err != nil {
		return err
	}
	return es.s.Put(normalize("endpoint", endpoint.EndpointID), data, nil)
}

func (es Endpoints) Delete(id string) error {
	return es.s.Delete(normalize("endpoint", id))
}
