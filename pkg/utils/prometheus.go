package utils

import (
	"context"
	"os"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/klog"
)

var NodeMetrics = []string{
	"node_cpu_utilisation",
	"node_memory_utilisation",
	"node_load1",
	"node_load5",
	// "node_load15",
}

var MetricsPromqlMap = map[string]string{
	"node_cpu_utilisation": `avg by (cluster,node) (irate(node_cpu_seconds_total{job="node-exporter",mode="used"}[5m]) * on 
	(cluster,namespace, pod) group_left(node) node_namespace_pod:kube_pod_info:)`,
	"node_memory_utilisation": `node:node_memory_utilisation:`,
	"node_load1":              `sum by (cluster,node) (node_load1{job="node-exporter"} * on (cluster,namespace, pod) group_left(node) node_namespace_pod:kube_pod_info:) / count by (cluster,node) (sum by (cluster,node, cpu) (node_cpu_seconds_total{job="node-exporter"} * on (cluster,namespace, pod) group_left(node) node_namespace_pod:kube_pod_info:))`,
	"node_load5":              `sum by (cluster,node) (node_load5{job="node-exporter"} * on (cluster,namespace, pod) group_left(node) node_namespace_pod:kube_pod_info:) / count by (cluster,node) (sum by (cluster,node, cpu) (node_cpu_seconds_total{job="node-exporter"} * on (cluster,namespace, pod) group_left(node) node_namespace_pod:kube_pod_info:))`,
	"node_load15":             `sum by (cluster,node) (node_load15{job="node-exporter"} * on (cluster,namespace, pod) group_left(node) node_namespace_pod:kube_pod_info:) / count by (cluster,node) (sum by (cluster,node, cpu) (node_cpu_seconds_total{job="node-exporter"} * on (cluster,namespace, pod) group_left(node) node_namespace_pod:kube_pod_info:))`,
}

//prometheus query 重构prometheus query
func QueryRebuild(v1api v1.API, query string, ts time.Time) (map[string]float64, bool) {
	ctx := context.Background()
	result, warnings, err := v1api.Query(ctx, query, ts)
	if err != nil {
		klog.Error("Error querying Prometheus: %s\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		klog.Error("Warnings: %v\n", warnings)
	}

	if result.String() != "" {
		// resultslice := strings.Split(result.String(), "\n")
		// resultSliceMap := ConvertResultDataType(resultslice)
		// return resultSliceMap, true

		resultMap := ConvertDataType(result)
		return resultMap, true
	} else {
		klog.Errorf("查询promql有问题,promsql返回结果为:nil,%v", result)
		return nil, false
	}
}

func ConvertDataType(result model.Value) map[string]float64 {
	resultMap := make(map[string]float64)

	switch result.Type().String() {
	case "vector":
		vec, ok := result.(model.Vector)
		if !ok {
			klog.Errorf("Convert result %s to vector type failed.", result.String())
		}
		for _, v := range vec {
			node := string(v.Metric["node"])
			resultMap[node] = float64(v.Value)
		}
	}
	return resultMap
}
