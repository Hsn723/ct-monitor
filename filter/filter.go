package filter

import (
	"os/exec"
	"time"

	"github.com/Hsn723/certspotter-client/api"
	"github.com/hashicorp/go-plugin"
)

// ApplyFilters runs the filter plugins and returns the resulting issuances.
// Plugin errors are ignored. It is up to the plugin to log them appropriately.
func ApplyFilters(filterPaths []string, issuances []api.Issuance) ([]api.Issuance, error) {
	res := issuances
	for _, fp := range filterPaths {
		r, err := applyFilter(fp, res)
		if err != nil {
			return res, err
		}
		res = r
	}
	return res, nil
}

func applyFilter(filter string, issuances []api.Issuance) ([]api.Issuance, error) {
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  HandshakeConfig,
		Plugins:          PluginMap,
		Cmd:              exec.Command(filter),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
		StartTimeout:     10 * time.Second,
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
