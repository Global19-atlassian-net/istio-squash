package envoy

import (
	"strings"

	"istio.io/istio/pilot/model"
)

type SquashConfig struct {
	SquashCluster string `json:"squash_cluster"`
}

func genSquashConfig(cluster *Cluster) *SquashConfig {
	return &SquashConfig{
		SquashCluster: cluster.Name,
	}
}
func isSquash(service *model.Service) bool {
	return strings.Contains(service.Hostname, "squash-server") || strings.Contains(service.Hostname, "squash-client")
}

func buildSquashFiltersClusters(instances *model.ServiceInstance, services []*model.Service) ([]HTTPFilter, Clusters) {
	if isSquash(instances.Service) {
		// don't add squash to instances of  squash
		return nil, nil
	}
	// find external services with http
	// create a cluster with the external host name as static dns destination
	// TODO create ssl context that verifies the destination SAN
	var filters []HTTPFilter
	var cs Clusters
	for _, service := range services {

		if !isSquash(service) {
			continue
		}
		destination := service.Hostname
		if port, ok := service.Ports.Get("http-squash-api"); ok {
			cluster := buildOutboundCluster(destination, port, nil)
			cs = append(cs, cluster)
			filters = append(filters, HTTPFilter{
				Type:   decoder,
				Name:   "squash",
				Config: genSquashConfig(cluster),
			})
			break
		}

	}

	return filters, cs
}
