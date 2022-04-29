package cmd

import (
	"io"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"dynamic-annotator/pkg/node"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

type serverRunOptions struct {
	clientSet *kubernetes.Clientset
	promAPI   promv1.API
}

func newOptions() serverRunOptions {
	return serverRunOptions{}
}

func complete(o *serverRunOptions) error {

	// init kubernetes clientset
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		klog.Errorf("Get config error: %v\n", err)
		return err
	}
	o.clientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		klog.Errorf("Get ClientSet error: %v\n", err)
		return err
	}

	// init prometheus client api
	client, err := api.NewClient(api.Config{
		Address: promAddr,
	})
	if err != nil {
		klog.Errorf("Error creating client: %v\n", err)
		return err
	}
	o.promAPI = promv1.NewAPI(client)

	return nil
}

func run(o *serverRunOptions) error {
	stopCh := make(chan struct{})

	// create informer factory
	informerFactory := informers.NewSharedInformerFactory(o.clientSet, 10*time.Second)

	if err := node.UpdateNodeByMetrics(stopCh, informerFactory, o.clientSet, o.promAPI, scrape_interval); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", Healyth)
	server := &http.Server{
		Addr:    webAddr,
		Handler: mux,
	}

	klog.Infof("web server started on %s.\n", webAddr)
	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func Healyth(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "ok")
}
