package controller

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/mirage20/lite-mesh/pkg/apis/mesh/v1alpha1"
	meshclientset "github.com/mirage20/lite-mesh/pkg/client/clientset/versioned"
	meshinformers "github.com/mirage20/lite-mesh/pkg/client/informers/externalversions/mesh/v1alpha1"
	meshlisters "github.com/mirage20/lite-mesh/pkg/client/listers/mesh/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsv1informers "k8s.io/client-go/informers/apps/v1"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type controller struct {
	kubeClient       kubernetes.Interface
	meshClient       meshclientset.Interface
	deploymentLister appsv1listers.DeploymentLister
	k8sServiceLister corev1listers.ServiceLister
	serviceLister    meshlisters.ServiceLister

	workqueue workqueue.RateLimitingInterface
}

func New(
	kubeClient kubernetes.Interface,
	meshClient meshclientset.Interface,
	deploymentInformer appsv1informers.DeploymentInformer,
	k8sServiceInformer corev1informers.ServiceInformer,
	serviceInformer meshinformers.ServiceInformer,
) *controller {
	c := &controller{
		kubeClient:       kubeClient,
		meshClient:       meshClient,
		deploymentLister: deploymentInformer.Lister(),
		k8sServiceLister: k8sServiceInformer.Lister(),
		serviceLister:    serviceInformer.Lister(),
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Service"),
	}

	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(old, new interface{}) {
			c.enqueue(new)
		},
	})

	return c
}

func (c *controller) Run(threadiness int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	glog.Info("Starting Service controller")

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	glog.Info("Shutting Service controller")

}

func (c *controller) enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		t := time.Now()
		// Run the handler, passing it the namespace/name string of the resource.
		if err := c.reconcile(key); err != nil {
			glog.Infoln("Sync failed", "key", key, "time", time.Since(t))
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		glog.Infoln("Successfully synced", "key", key, "time", time.Since(t))
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *controller) reconcile(key string) error {

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		glog.Errorf("invalid resource key: %s", key)
		return nil
	}
	serviceOriginal, err := c.serviceLister.Services(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("service '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	glog.Infoln("lister instance", key, serviceOriginal)
	service := serviceOriginal.DeepCopy()

	if err := c.reconcileDeployment(service); err != nil {
		return err
	}

	if len(service.Spec.Ports()) > 0 || service.Spec.Gateway != nil {
		if err := c.reconcileK8sService(service); err != nil {
			return err
		}
	}
	return nil
}

func (c *controller) reconcileDeployment(service *v1alpha1.Service) error {

	deployment, err := c.deploymentLister.Deployments(service.Namespace).Get(deploymentName(service))
	if errors.IsNotFound(err) {
		deployment, err = c.kubeClient.AppsV1().Deployments(service.Namespace).Create(CreateServiceDeployment(service))
		if err != nil {
			glog.Errorf("Failed to create TokenService Deployment %v", err)
			return err
		}
		glog.Infoln("Deployment created", deployment)
	} else if err != nil {
		return err
	}

	return nil
}

func (c *controller) reconcileK8sService(service *v1alpha1.Service) error {

	k8sService, err := c.k8sServiceLister.Services(service.Namespace).Get(k8sServiceName(service))
	if errors.IsNotFound(err) {
		k8sService, err = c.kubeClient.CoreV1().Services(service.Namespace).Create(CreateServiceK8sService(service))
		if err != nil {
			glog.Errorf("Failed to create TokenService service %v", err)
			return err
		}
		glog.Infoln("Service created", k8sService)
	} else if err != nil {
		return err
	}
	return nil
}
