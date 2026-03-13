package monitoring

// QueryTemplates contains named PromQL query templates for each resource type.
// Variable values are validated against Kubernetes name regex before substitution.
var QueryTemplates = map[string]QueryTemplate{
	// Pod metrics
	"pod_cpu_usage": {
		Name:        "pod_cpu_usage",
		Description: "CPU usage in cores per container",
		Query:       `sum(rate(container_cpu_usage_seconds_total{container!="",pod="$pod",namespace="$namespace"}[5m])) by (container)`,
		Variables:   []string{"namespace", "pod"},
	},
	"pod_memory_usage": {
		Name:        "pod_memory_usage",
		Description: "Memory working set bytes per container",
		Query:       `sum(container_memory_working_set_bytes{container!="",pod="$pod",namespace="$namespace"}) by (container)`,
		Variables:   []string{"namespace", "pod"},
	},
	"pod_network_rx": {
		Name:        "pod_network_rx",
		Description: "Network receive bytes per second",
		Query:       `sum(rate(container_network_receive_bytes_total{pod="$pod",namespace="$namespace"}[5m]))`,
		Variables:   []string{"namespace", "pod"},
	},
	"pod_network_tx": {
		Name:        "pod_network_tx",
		Description: "Network transmit bytes per second",
		Query:       `sum(rate(container_network_transmit_bytes_total{pod="$pod",namespace="$namespace"}[5m]))`,
		Variables:   []string{"namespace", "pod"},
	},

	// Node metrics
	"node_cpu_utilization": {
		Name:        "node_cpu_utilization",
		Description: "Node CPU utilization percentage",
		Query:       `100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle",instance=~"$node.*"}[5m])) * 100)`,
		Variables:   []string{"node"},
	},
	"node_memory_utilization": {
		Name:        "node_memory_utilization",
		Description: "Node memory utilization percentage",
		Query:       `100 * (1 - node_memory_MemAvailable_bytes{instance=~"$node.*"} / node_memory_MemTotal_bytes{instance=~"$node.*"})`,
		Variables:   []string{"node"},
	},
	"node_disk_utilization": {
		Name:        "node_disk_utilization",
		Description: "Node root disk utilization percentage",
		Query:       `100 - (node_filesystem_avail_bytes{instance=~"$node.*",mountpoint="/",fstype!="rootfs"} / node_filesystem_size_bytes{instance=~"$node.*",mountpoint="/",fstype!="rootfs"} * 100)`,
		Variables:   []string{"node"},
	},
	"node_pod_count": {
		Name:        "node_pod_count",
		Description: "Number of pods on the node",
		Query:       `count(kube_pod_info{node="$node"})`,
		Variables:   []string{"node"},
	},

	// Deployment metrics
	"deployment_replica_health": {
		Name:        "deployment_replica_health",
		Description: "Ratio of available to desired replicas",
		Query:       `kube_deployment_status_replicas_available{namespace="$namespace",deployment="$deployment"} / kube_deployment_spec_replicas{namespace="$namespace",deployment="$deployment"}`,
		Variables:   []string{"namespace", "deployment"},
	},

	// PVC metrics
	"pvc_usage_percent": {
		Name:        "pvc_usage_percent",
		Description: "PVC storage usage percentage",
		Query:       `kubelet_volume_stats_used_bytes{namespace="$namespace",persistentvolumeclaim="$pvc"} / kubelet_volume_stats_capacity_bytes{namespace="$namespace",persistentvolumeclaim="$pvc"} * 100`,
		Variables:   []string{"namespace", "pvc"},
	},
}

// ResourceDashboardMap maps resource kinds to their Grafana dashboard UIDs
// and template variable names.
var ResourceDashboardMap = map[string]struct {
	UID     string
	VarName string
}{
	"pods":                    {UID: "kubecenter-pod-detail", VarName: "pod"},
	"deployments":            {UID: "kubecenter-deployment-detail", VarName: "deployment"},
	"statefulsets":            {UID: "kubecenter-statefulset-detail", VarName: "statefulset"},
	"daemonsets":              {UID: "kubecenter-daemonset-detail", VarName: "daemonset"},
	"nodes":                  {UID: "kubecenter-node-detail", VarName: "node"},
	"persistentvolumeclaims": {UID: "kubecenter-pvc-detail", VarName: "pvc"},
}
