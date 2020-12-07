package pusher

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	// IssuancesObserved records certificate issuances
	IssuancesObserved = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "issuances_observed"},
		[]string{"id", "domain", "dns_names", "issuer", "not_before"},
	)
)

// GetPusher initializes and returns a pusher for the given URL
func GetPusher(url string) *push.Pusher {
	registry := prometheus.NewRegistry()
	registry.MustRegister(IssuancesObserved)
	return push.New(url, "ct_monitor").Gatherer(registry)
}
