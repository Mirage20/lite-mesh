package controller

import (
	"github.com/mirage20/lite-mesh/pkg/apis/mesh/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func deploymentName(service *v1alpha1.Service) string {
	return service.Name
}

func k8sServiceName(service *v1alpha1.Service) string {
	return service.Name
}


func createLabels(service *v1alpha1.Service) map[string]string {
	labels := make(map[string]string, len(service.ObjectMeta.Labels)+1)
	labels["app"] = service.Name
	for k, v := range service.ObjectMeta.Labels {
		labels[k] = v
	}
	return labels
}

func createSelector(service *v1alpha1.Service) *metav1.LabelSelector {
	return &metav1.LabelSelector{MatchLabels: createLabels(service)}
}

func createServiceOwnerRef(obj metav1.Object) *metav1.OwnerReference {
	return metav1.NewControllerRef(obj, schema.GroupVersionKind{
		Group:   v1alpha1.SchemeGroupVersion.Group,
		Version: v1alpha1.SchemeGroupVersion.Version,
		Kind:    "Service",
	})
}
