package main

import (
	"flag"
	"fmt"
	"github.com/mirage20/lite-mesh/pkg/xds"
	"log"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	meshclientset "github.com/mirage20/lite-mesh/pkg/client/clientset/versioned"
	meshinformers "github.com/mirage20/lite-mesh/pkg/client/informers/externalversions"
	"github.com/mirage20/lite-mesh/pkg/signals"
)

const (
	threadsPerController = 2
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	flag.Parse()

	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	meshClient, err := meshclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building mesh clientset: %s", err.Error())
	}

	// Create informer factories
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*10)
	meshInformerFactory := meshinformers.NewSharedInformerFactory(meshClient, time.Second*10)

	// Create informers
	podInformer := kubeInformerFactory.Core().V1().Pods()
	configurationInformer := meshInformerFactory.Mesh().V1alpha1().Configurations()


	server := xds.NewServer(configurationInformer, podInformer)

	// Start informers
	go kubeInformerFactory.Start(stopCh)
	go meshInformerFactory.Start(stopCh)

	// Wait for cache sync
	log.Println("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh,
		podInformer.Informer().HasSynced,
		configurationInformer.Informer().HasSynced,
	); !ok {
		log.Fatal("failed to wait for caches to sync")
	}

	//Start XDS server
	go server.Start(stopCh)


	fmt.Println("started")
	// Prevent exiting the main process
	<-stopCh
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
