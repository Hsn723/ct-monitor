package filter

import (
	"time"

	"github.com/Hsn723/certspotter-client/api"
)

const (
	pluginTimeout = 10 * time.Second
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
	if isLuaFilter(filter) {
		return applyLuaFilter(filter, issuances)
	}
	return applyRPCFilter(filter, issuances)
}
