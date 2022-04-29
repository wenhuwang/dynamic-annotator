package node

import (
	"encoding/json"
	"fmt"
	"time"

	"dynamic-annotator/pkg/utils"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

func UpdateNodeByMetrics(stopCh <-chan struct{}, factory informers.SharedInformerFactory,
	clientset *kubernetes.Clientset, v1api promv1.API, interval int) error {

	// start node informer
	nodeInformer := factory.Core().V1().Nodes()
	go nodeInformer.Informer().Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, nodeInformer.Informer().HasSynced) {
		return fmt.Errorf("Timed out waiting for caches to sync")
	}

	metricsAnnotationsMap := map[string]float64{
		"node_cpu_utilisation": 0,
	}

	klog.Info("Start sync node metrics.")

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// query all metrics
			monitorResult := map[string]map[string]float64{}
			for _, metricsName := range utils.NodeMetrics {
				result, ok := utils.QueryRebuild(v1api, utils.MetricsPromqlMap[metricsName], time.Now())
				if !ok {
					klog.Warningf("Query metrics %s failed.\n", metricsName)
					continue
				}
				monitorResult[metricsName] = result
			}

			// combine annotations data and update to the node
			nodes, err := nodeInformer.Lister().List(labels.NewSelector())
			if err != nil {
				klog.Errorf("NodeInformer list failed. %v\n", err)
				continue
			}
			for _, node := range nodes {
				// if node.Name != "10.165.5.27" {
				// 	continue
				// }
				for _, metricsName := range utils.NodeMetrics {
					metricsAnnotationsMap[metricsName] = monitorResult[metricsName][node.GetName()]
				}
				// fmt.Printf("metricsAnnotationsMap is %v\n", metricsAnnotationsMap)

				metricsAnnotationsString, _ := json.Marshal(metricsAnnotationsMap)
				patchData := map[string]interface{}{
					"metadata": map[string]map[string]string{
						"annotations": {
							"metrics.kubernetes.io": string(metricsAnnotationsString),
						},
					},
				}
				patchBytes, err := json.Marshal(patchData)
				if err != nil {
					klog.Errorf("Patchdata json marshal failed. %v\n", err)
					continue
				}
				if _, err := clientset.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType, patchBytes); err != nil {
					klog.Errorf("Patch node %s failed. %v\n", node.Name, err)
					continue
				}
			}
		}
	}()
	return nil
}
