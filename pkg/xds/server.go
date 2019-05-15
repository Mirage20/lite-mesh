package xds

import (
	"bytes"
	"fmt"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	fileconf "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	xdscache "github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/types"
	"github.com/mirage20/lite-mesh/pkg/apis/mesh/v1alpha1"
	meshinformers "github.com/mirage20/lite-mesh/pkg/client/informers/externalversions/mesh/v1alpha1"
	meshlisters "github.com/mirage20/lite-mesh/pkg/client/listers/mesh/v1alpha1"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/labels"
	coreinformers "k8s.io/client-go/informers/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"time"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	v21 "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	httpproxy "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	tcpproxy "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	"github.com/gogo/protobuf/proto"
	"net"
	"strings"
)

type server struct {
	configurationLister meshlisters.ConfigurationLister
	podLister           corelisters.PodLister
	snapshotCache       xdscache.SnapshotCache
}

func NewServer(configurationInformer meshinformers.ConfigurationInformer, podInformer coreinformers.PodInformer) *server {

	s := &server{
		configurationLister: configurationInformer.Lister(),
		podLister:           podInformer.Lister(),
	}
	s.snapshotCache = xdscache.NewSnapshotCache(true, s, nil)

	configurationInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: s.update,
		UpdateFunc: func(old, new interface{}) {
			s.update(new)
		},
	})

	return s
}

func (s *server) Start(stopCh <-chan struct{}) {
	server := xds.NewServer(s.snapshotCache, nil)
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":9000")

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)

	if err := grpcServer.Serve(lis); err != nil {
	}
}

func (s *server) ID(node *core.Node) string {
	//fmt.Printf("%+v", node)
	//strings.Split(node.Id,"@")[1]
	return strings.Split(node.Id, "@")[1]
}

func (s *server) update(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		return
	}
	fmt.Println(key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		fmt.Println(err)
		return
	}
	configurationsOriginal, err := s.configurationLister.Configurations(namespace).Get(name)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, rule := range configurationsOriginal.Spec.Rules {
		pods, err := s.podLister.List(labels.SelectorFromSet(rule.Match))
		if err != nil {
			fmt.Println(err)
		}
		for _, pod := range pods {
			var endpoints, routes []xdscache.Resource
			snapshot := xdscache.NewSnapshot(configurationsOriginal.ResourceVersion,
				endpoints,
				buildClusters(rule.Clusters),
				routes,
				buildListeners(rule.Filters),
			)
			err = s.snapshotCache.SetSnapshot(pod.Name, snapshot)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func buildListeners(filters []v1alpha1.Filter) []xdscache.Resource {
	var listeners []xdscache.Resource

	var fchain []listener.FilterChain

	fchain = append(fchain, listener.FilterChain{
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
	})

	for _, filter := range filters {
		chain := listener.FilterChain{
			FilterChainMatch: &listener.FilterChainMatch{
				DestinationPort: &types.UInt32Value{
					Value: filter.Port,
				},
			},
		}
		if len(filter.Http) > 0 {

			var vhosts []route.VirtualHost

			for i, v := range filter.Http {
				vhosts = append(vhosts, route.VirtualHost{
					Name:    fmt.Sprintf("http-%d-%d", filter.Port, i),
					Domains: v.Domains,
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
										Cluster: v.Cluster,
									},
								},
							},
						},
					},
				})
			}

			chain.Filters = []listener.Filter{
				{
					Name: "envoy.http_connection_manager",
					ConfigType: &listener.Filter_Config{
						Config: MessageToStruct(&httpproxy.HttpConnectionManager{
							StatPrefix: fmt.Sprintf("http-%d", filter.Port),
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
									Name:         fmt.Sprintf("http-route-%d", filter.Port),
									VirtualHosts: vhosts,
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
			}

		} else {
			chain.Filters = []listener.Filter{
				{
					Name: "envoy.tcp_proxy",
					ConfigType: &listener.Filter_Config{
						Config: MessageToStruct(&tcpproxy.TcpProxy{
							StatPrefix: fmt.Sprintf("tcp-%d", filter.Port),
							ClusterSpecifier: &tcpproxy.TcpProxy_Cluster{
								Cluster: filter.Tcp.Cluster,
							},
						}),
					},
				},
			}
		}
		fchain = append(fchain, chain)
	}

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
		FilterChains: fchain,
	})
	return listeners
}

func buildClusters(clusters []v1alpha1.Cluster) []xdscache.Resource {
	var envoyClusters []xdscache.Resource

	for _, cluster := range clusters {
		envoyClusters = append(envoyClusters, &api.Cluster{
			Name:           cluster.Name,
			ConnectTimeout: time.Second,
			ClusterDiscoveryType: &api.Cluster_Type{
				Type: api.Cluster_STRICT_DNS,
			},
			Hosts: []*core.Address{
				{
					Address: &core.Address_SocketAddress{
						SocketAddress: &core.SocketAddress{
							Protocol: core.TCP,
							Address:  cluster.Host,
							PortSpecifier: &core.SocketAddress_PortValue{
								PortValue: cluster.Port,
							},
						},
					},
				},
			},
			DnsLookupFamily: api.Cluster_V4_ONLY,
		})
	}

	return envoyClusters
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
