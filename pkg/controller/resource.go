package controller

import (
	"fmt"
	"github.com/mirage20/lite-mesh/pkg/apis/mesh/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func CreateServiceDeployment(service *v1alpha1.Service) *appsv1.Deployment {
	podTemplateAnnotations := map[string]string{}
	podTemplateAnnotations["sidecar.istio.io/inject"] = "false"

	if len(service.Spec.Container.Name) == 0 {
		service.Spec.Container.Name = service.Name
	}

	var containers []corev1.Container

	containers = append(containers, corev1.Container{
		Args: []string{
			"--bootstrapTemplate",
			"/etc/conf/envoy-bootstrap-template.yaml",
			"--bootstrapConfig",
			"/etc/conf/envoy-bootstrap.yaml",
			"--envoyBinary",
			"/usr/local/bin/envoy",
			"--logLevel",
			envoyLogLevel(service),
			"--serviceCluster",
			service.Name,
			"--discoveryAddress",
			envoyDiscoveryAddress(service),
			"--discoveryPort",
			envoyDiscoveryPort(service),
		},
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
					},
				},
			},
			{
				Name: "POD_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "status.podIP",
					},
				},
			},
		},
		Image: "mirage20/lite-mesh-envoy-bootstrap",
		Name:  "envoy-proxy",
	})

	if service.Spec.Gateway == nil {
		containers = append(containers, service.Spec.Container)
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName(service),
			Namespace: service.Namespace,
			Labels:    createLabels(service),
			OwnerReferences: []metav1.OwnerReference{
				*createServiceOwnerRef(service),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: service.Spec.Replicas,
			Selector: createSelector(service),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      createLabels(service),
					Annotations: podTemplateAnnotations,
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Args: []string{
								"-p",
								"15001",
								"-u",
								"2020",
								"-m",
								"REDIRECT",
								"-i",
								"*",
								"-x",
								"",
								"-b",
								"*",
								"-d",
								"",
							},
							Image: "gcr.io/istio-release/proxy_init:1.0.2",
							Name:  "iptable-init",
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"NET_ADMIN",
									},
								},
							},
						},
					},
					Containers: containers,
				},
			},
		},
	}
}

func CreateServiceK8sService(service *v1alpha1.Service) *corev1.Service {

	var servicePorts []corev1.ServicePort

	if service.Spec.Gateway != nil {
		for _, v := range service.Spec.Gateway.Ports {
			servicePorts = append(servicePorts, corev1.ServicePort{
				Name:       fmt.Sprintf("tcp-%d", v),
				Protocol:   corev1.ProtocolTCP,
				Port:       v,
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: v},
			})
		}
	} else {
		for _, v := range service.Spec.Ports() {
			servicePorts = append(servicePorts, corev1.ServicePort{
				Name:       fmt.Sprintf("tcp-%d", v),
				Protocol:   corev1.ProtocolTCP,
				Port:       v,
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: v},
			})
		}
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sServiceName(service),
			Namespace: service.Namespace,
			Labels:    createLabels(service),
			OwnerReferences: []metav1.OwnerReference{
				*createServiceOwnerRef(service),
			},
		},
		Spec: corev1.ServiceSpec{
			Ports:    servicePorts,
			Selector: createLabels(service),
		},
	}
}

func envoyLogLevel(service *v1alpha1.Service) string {
	if len(service.Spec.Envoy.LogLevel) > 0 {
		return service.Spec.Envoy.LogLevel
	}
	return "info"
}

func envoyDiscoveryAddress(service *v1alpha1.Service) string {
	if len(service.Spec.Envoy.DiscoveryAddress) > 0 {
		return service.Spec.Envoy.DiscoveryAddress
	}
	return "10.100.5.46"
}

func envoyDiscoveryPort(service *v1alpha1.Service) string {
	if len(service.Spec.Envoy.DiscoveryPort) > 0 {
		return service.Spec.Envoy.DiscoveryPort
	}
	return "9000"
}
