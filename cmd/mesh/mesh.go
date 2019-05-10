package main

import (
	"bytes"
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	fileconf "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	v21 "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	httpproxy "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	tcpproxy "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"google.golang.org/grpc"
	"net"
	"time"
)

type hash struct {
}

type stdlog struct {
}

func main() {
	snapshotCache := cache.NewSnapshotCache(true, &hash{}, &stdlog{})
	server := xds.NewServer(snapshotCache, nil)
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":9000")

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	//api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	//api.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	//api.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	//api.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	//go func() {

	//id := "sidecar~192.168.0.132~echo-service-5d7bdb498d-ckk8q.default~default.svc.cluster.local"
	ticker := time.NewTicker(5 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				fmt.Println("============ Start =================")
				for _, v := range snapshotCache.GetStatusKeys() {
					var clusters, endpoints, routes, listeners []cache.Resource

					listeners = append(listeners, &api.Listener{
						Name: "virtual",
						Address: core.Address{
							Address: &core.Address_SocketAddress{
								SocketAddress: &core.SocketAddress{
									Protocol: core.TCP,
									Address:  "0.0.0.0",
									PortSpecifier: &core.SocketAddress_PortValue{
										PortValue: 15001,
									},
								},
							},
						},
						ListenerFilters: []listener.ListenerFilter{
							{
								Name: "envoy.listener.original_dst",
							},
						},
						FilterChains: []listener.FilterChain{
							{
								Filters: []listener.Filter{
									{
										Name: "envoy.tcp_proxy",
										ConfigType: &listener.Filter_Config{
											Config: MessageToStruct(&tcpproxy.TcpProxy{
												StatPrefix: "BlackHoleCluster",
												ClusterSpecifier: &tcpproxy.TcpProxy_Cluster{
													Cluster: "BlackHoleCluster",
												},
											}),
										},
									},
								},
							},
							{
								FilterChainMatch: &listener.FilterChainMatch{
									DestinationPort: &types.UInt32Value{
										Value: 80,
									},
								},
								Filters: []listener.Filter{
									{
										Name: "envoy.http_connection_manager",
										ConfigType: &listener.Filter_Config{
											Config: MessageToStruct(&httpproxy.HttpConnectionManager{
												StatPrefix: "ingress_http",
												AccessLog: []*v21.AccessLog{
													{
														Name: "envoy.file_access_log",
														ConfigType: &v21.AccessLog_Config{
															Config: MessageToStruct(&fileconf.FileAccessLog{
																Path: "/dev/stdout",
															}),
														},
													},
												},
												RouteSpecifier: &httpproxy.HttpConnectionManager_RouteConfig{
													RouteConfig: &v2.RouteConfiguration{
														Name: "local_route",
														VirtualHosts: []route.VirtualHost{
															{
																Name: "local_route",
																Domains: []string{
																	"google.com",
																	"example.com",
																	"*.facebook.com",
																},
																Routes: []route.Route{
																	{
																		Match: route.RouteMatch{
																			PathSpecifier: &route.RouteMatch_Prefix{
																				Prefix: "/",
																			},
																		},
																		Action: &route.Route_Route{
																			Route: &route.RouteAction{
																				ClusterSpecifier: &route.RouteAction_Cluster{
																					Cluster: "service_example",
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
												HttpFilters: []*httpproxy.HttpFilter{
													{
														Name: "envoy.router",
													},
												},
											}),
										},
									},
								},
							},
							{
								FilterChainMatch: &listener.FilterChainMatch{
									DestinationPort: &types.UInt32Value{
										Value: 8080,
									},
								},
								Filters: []listener.Filter{
									{
										Name: "envoy.tcp_proxy",
										ConfigType: &listener.Filter_Config{
											Config: MessageToStruct(&tcpproxy.TcpProxy{
												StatPrefix: "ingress_tcp",
												ClusterSpecifier: &tcpproxy.TcpProxy_Cluster{
													Cluster: "mesh_service",
												},
											}),
										},
									},
								},
							},
						},
					})

					clusters = append(clusters, &api.Cluster{
						Name:           "service_example",
						ConnectTimeout: time.Second,
						ClusterDiscoveryType: &api.Cluster_Type{
							Type: api.Cluster_STATIC,
						},
						Hosts: []*core.Address{
							{
								Address: &core.Address_SocketAddress{
									SocketAddress: &core.SocketAddress{
										Protocol: core.TCP,
										Address:  "93.184.216.34",
										PortSpecifier: &core.SocketAddress_PortValue{
											PortValue: 80,
										},
									},
								},
							},
						},
					})

					clusters = append(clusters, &api.Cluster{
						Name:           "mesh_service",
						ConnectTimeout: time.Second,
						ClusterDiscoveryType: &api.Cluster_Type{
							Type: api.Cluster_STRICT_DNS,
						},
						Hosts: []*core.Address{
							{
								Address: &core.Address_SocketAddress{
									SocketAddress: &core.SocketAddress{
										Protocol: core.TCP,
										Address:  "mesh-server",
										PortSpecifier: &core.SocketAddress_PortValue{
											PortValue: 8080,
										},
									},
								},
							},
						},
					})
					snapshot := cache.NewSnapshot("1.5", endpoints, clusters, routes, listeners)
					_ = snapshotCache.SetSnapshot(v, snapshot)
					fmt.Println(snapshotCache.GetSnapshot(v))
				}
				fmt.Println("============= End ==================")
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	if err := grpcServer.Serve(lis); err != nil {
		// error handling
	}
	//}()
}

func Any(pb proto.Message) *types.Any {
	m, err := types.MarshalAny(&api.Listener{
		Name: "virtual",
	})
	if err != nil {
		fmt.Println("errrrrrrrrrrrrrrrrr: ", err)
	}
	l, _ := m.Marshal()
	x := api.Listener{}
	x.Unmarshal(l)
	fmt.Println(x)
	return m
}

func MessageToStruct(msg proto.Message) *types.Struct {

	buf := &bytes.Buffer{}
	if err := (&jsonpb.Marshaler{OrigName: true}).Marshal(buf, msg); err != nil {
		return nil
	}

	pbs := &types.Struct{}
	if err := jsonpb.Unmarshal(buf, pbs); err != nil {
		return nil
	}

	return pbs
}

func (h *hash) ID(node *core.Node) string {
	fmt.Printf("%+v", node)
	return node.Id
}

func (log *stdlog) Infof(format string, args ...interface{}) {
	fmt.Printf(fmt.Sprintf("Infoooooooo %s\n", format), args)
}

func (log *stdlog) Errorf(format string, args ...interface{}) {
	fmt.Printf(fmt.Sprintf("Errorrrrrrr %s\n", format), args)
}
