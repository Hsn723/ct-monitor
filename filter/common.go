package filter

import (
	"github.com/Hsn723/certspotter-client/api"
	"github.com/hashicorp/go-plugin"
	"net/rpc"
)

var (
	HandshakeConfig = plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "CT_MONITOR_PLUGIN",
		MagicCookieValue: "issuance_filter",
	}
	PluginKey = "issuance"
	PluginMap = map[string]plugin.Plugin{
		PluginKey: &IssuanceFilterPlugin{},
	}
)

// IssuanceFilter is the interface exposed as a plugin.
type IssuanceFilter interface {
	Filter(is []api.Issuance) ([]api.Issuance, error)
}

// IssuanceFilterRPC is a plugin implementation over RPC.
type IssuanceFilterRPCClient struct {
	client *rpc.Client
}

func (f *IssuanceFilterRPCClient) Filter(is []api.Issuance) ([]api.Issuance, error) {
	var resp []api.Issuance
	err := f.client.Call("Plugin.Filter", is, &resp)
	return resp, err
}

// IssuanceFilterRPCServer is the RPC server that IssuanceFilterRPC talks to.
type IssuanceFilterRPCServer struct {
	Impl IssuanceFilter
}

func (s *IssuanceFilterRPCServer) Filter(is []api.Issuance, resp *[]api.Issuance) error {
	r, err := s.Impl.Filter(is)
	*resp = r
	return err
}

// IssuanceFilterPlugin us an implementation of plugin.Plugin.
type IssuanceFilterPlugin struct {
	Impl IssuanceFilter
}

func (p *IssuanceFilterPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &IssuanceFilterRPCServer{Impl: p.Impl}, nil
}

func (*IssuanceFilterPlugin) Client(_ *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &IssuanceFilterRPCClient{client: c}, nil
}
