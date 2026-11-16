//go:build testfilter
// +build testfilter

package main

import (
	"log/slog"

	"github.com/Hsn723/certspotter-client/api"
	"github.com/Hsn723/ct-monitor/filter"
	"github.com/hashicorp/go-plugin"
)

type sampleFilter struct{}

func (sampleFilter) Filter(is []api.Issuance) ([]api.Issuance, error) {
	slog.Info("running sample filter", "issuances", len(is))
	if len(is) > 1 {
		return is[:1], nil
	}
	return is, nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: filter.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			filter.PluginKey: &filter.IssuanceFilterPlugin{Impl: &sampleFilter{}},
		},
	})
}
