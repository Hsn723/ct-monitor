//go:build testfilter
// +build testfilter

package main

import (
	"github.com/Hsn723/certspotter-client/api"
	"github.com/Hsn723/ct-monitor/filter"
	"github.com/cybozu-go/log"
	"github.com/hashicorp/go-plugin"
)

type sampleFilter struct{}

func (sampleFilter) Filter(is []api.Issuance) ([]api.Issuance, error) {
	_ = log.Info("running sample filter", map[string]interface{}{
		"issuances": len(is),
	})
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
