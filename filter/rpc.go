package filter

import (
	"os/exec"

	"github.com/Hsn723/certspotter-client/api"
	"github.com/hashicorp/go-plugin"
)

func applyRPCFilter(filter string, issuances []api.Issuance) ([]api.Issuance, error) {
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  HandshakeConfig,
		Plugins:          PluginMap,
		Cmd:              exec.Command(filter),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
		StartTimeout:     pluginTimeout,
		Managed:          true,
	})
	defer client.Kill()

	rpcClient, err := client.Client()
	if err != nil {
		return issuances, err
	}

	raw, err := rpcClient.Dispense(PluginKey)
	if err != nil {
		return issuances, err
	}

	issuanceFilter := raw.(IssuanceFilter)
	return issuanceFilter.Filter(issuances)
}
