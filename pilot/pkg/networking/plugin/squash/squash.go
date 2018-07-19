// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package squash

import (
	"strings"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	pbtypes "github.com/gogo/protobuf/types"

	squashconfig "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/squash/v2"
	http_conn "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/networking/plugin"
	"istio.io/istio/pilot/pkg/networking/util"
)

type squashplugin struct{}

// NewPlugin returns an ptr to an initialized envoyfilter.Plugin.
func NewPlugin() plugin.Plugin {
	return squashplugin{}
}

// OnOutboundListener implements the Callbacks interface method.
func (squashplugin) OnOutboundListener(in *plugin.InputParams, mutable *plugin.MutableObjects) error {
	return nil
}

// OnInboundListener implements the Callbacks interface method.
func (squashplugin) OnInboundListener(in *plugin.InputParams, mutable *plugin.MutableObjects) error {

	if in.ListenerType != plugin.ListenerTypeHTTP {
		return nil
	}

	// find a squash cluster:
	svcs, err := in.Env.ServiceDiscovery.Services()
	if err != nil {
		return err
	}

	service := getSquashSvc(svcs)
	if service == nil {
		return nil
	}

	if len(service.Ports) != 1 {
		return nil
	}

	// get cluster name from service name
	clustername := model.BuildSubsetKey(model.TrafficDirectionOutbound, "", service.Hostname, service.Ports[0].Port)

	squashConfig := squashconfig.Squash{
		Cluster: clustername,
		AttachmentTemplate: &pbtypes.Struct{
			Fields: map[string]*pbtypes.Value{
				"spec": &pbtypes.Value{
					Kind: &pbtypes.Value_StructValue{
						StructValue: &pbtypes.Struct{
							Fields: map[string]*pbtypes.Value{
								"match_request": &pbtypes.Value{
									Kind: &pbtypes.Value_BoolValue{
										BoolValue: true,
									},
								},
								"attachment": &pbtypes.Value{
									Kind: &pbtypes.Value_StructValue{
										StructValue: &pbtypes.Struct{
											Fields: map[string]*pbtypes.Value{
												"pod": &pbtypes.Value{
													Kind: &pbtypes.Value_StringValue{
														StringValue: "{{ POD_NAME }}",
													},
												},
												"namespace": &pbtypes.Value{
													Kind: &pbtypes.Value_StringValue{
														StringValue: "{{ POD_NAMESPACE }}",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	f := &http_conn.HttpFilter{
		Name:   "envoy.squash",
		Config: util.MessageToStruct(&squashConfig),
	}
	// add and configure the squash filter
	for cnum := range mutable.FilterChains {
		insertHTTPFilter(&mutable.FilterChains[cnum], f)
	}

	return nil
}

// OnOutboundCluster implements the Plugin interface method.
func (squashplugin) OnOutboundCluster(env model.Environment, node model.Proxy, service *model.Service, servicePort *model.Port, cluster *xdsapi.Cluster) {
	// do nothing
}

// OnInboundCluster implements the Plugin interface method.
func (squashplugin) OnInboundCluster(env model.Environment, node model.Proxy, service *model.Service, servicePort *model.Port, cluster *xdsapi.Cluster) {
	// do nothing
}

// OnOutboundRouteConfiguration implements the Plugin interface method.
func (squashplugin) OnOutboundRouteConfiguration(in *plugin.InputParams, routeConfiguration *xdsapi.RouteConfiguration) {
	// do nothing
}

// OnInboundRouteConfiguration implements the Plugin interface method.
func (squashplugin) OnInboundRouteConfiguration(in *plugin.InputParams, routeConfiguration *xdsapi.RouteConfiguration) {
	// do nothing
}

func insertHTTPFilter(filterChain *plugin.FilterChain, filter *http_conn.HttpFilter) {
	// insert last
	filterChain.HTTP = append(filterChain.HTTP, filter)
}

func getSquashSvc(svcs []*model.Service) *model.Service {

	for _, svc := range svcs {
		if strings.Contains(svc.Hostname.String(), "squash") {
			return svc
		}
	}
	return nil
}
