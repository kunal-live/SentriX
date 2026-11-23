package containers

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DockerContainer struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Image            string    `json:"image"`
	Status           string    `json:"status"` // RUNNING, PAUSED, EXITED, RESTARTING
	ServerID         string    `json:"server_id"`
	ServerName       string    `json:"server_name"`
	CPUUsagePct      float64   `json:"cpu_usage_pct"`
	MemoryUsageBytes uint64    `json:"memory_usage_bytes"`
	MemoryLimitBytes uint64    `json:"memory_limit_bytes"`
	NetRxBytes       uint64    `json:"net_rx_bytes"`
	NetTxBytes       uint64    `json:"net_tx_bytes"`
	RestartCount     int       `json:"restart_count"`
	OOMKilled        bool      `json:"oom_killed"`
	CreatedAt        time.Time `json:"created_at"`
}

type K8sClusterOverview struct {
	ClusterName          string    `json:"cluster_name"`
	Version              string    `json:"version"`
	Status               string    `json:"status"`
	TotalNodes           int       `json:"total_nodes"`
	ReadyNodes           int       `json:"ready_nodes"`
	TotalPods            int       `json:"total_pods"`
	RunningPods          int       `json:"running_pods"`
	PendingPods          int       `json:"pending_pods"`
	FailedPods           int       `json:"failed_pods"`
	TotalCPUMilli        int64     `json:"total_cpu_milli"`
	AllocatedCPUMilli    int64     `json:"allocated_cpu_milli"`
	TotalMemoryBytes     uint64    `json:"total_memory_bytes"`
	AllocatedMemoryBytes uint64    `json:"allocated_memory_bytes"`
	LastSyncAt           time.Time `json:"last_sync_at"`
	IsLiveCollector      bool      `json:"is_live_collector"`
}

type K8sNode struct {
	Name           string            `json:"name"`
	Role           string            `json:"role"`
	Status         string            `json:"status"` // Ready, NotReady
	InternalIP     string            `json:"internal_ip"`
	OSImage        string            `json:"os_image"`
	KubeletVersion string            `json:"kubelet_version"`
	CPUUsagePct    float64           `json:"cpu_usage_pct"`
	MemoryUsagePct float64           `json:"memory_usage_pct"`
	MemoryPressure bool              `json:"memory_pressure"`
	DiskPressure   bool              `json:"disk_pressure"`
	PIDPressure    bool              `json:"pid_pressure"`
	PodCount       int               `json:"pod_count"`
	PodCapacity    int               `json:"pod_capacity"`
	Labels         map[string]string `json:"labels"`
}

type K8sPod struct {
	Name          string    `json:"name"`
	Namespace     string    `json:"namespace"`
	Node          string    `json:"node"`
	Phase         string    `json:"phase"` // Running, Pending, Succeeded, Failed, CrashLoopBackOff
	RestartCount  int       `json:"restart_count"`
	OOMKilled     bool      `json:"oom_killed"`
	IP            string    `json:"ip"`
	Image         string    `json:"image"`
	CPUUsagePct   float64   `json:"cpu_usage_pct"`
	CPULimitMilli int64     `json:"cpu_limit_milli"`
	MemUsageMB    float64   `json:"mem_usage_mb"`
	MemLimitMB    float64   `json:"mem_limit_mb"`
	StatusReason  string    `json:"status_reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type K8sDeployment struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	DesiredReplicas   int    `json:"desired_replicas"`
	AvailableReplicas int    `json:"available_replicas"`
	UpdatedReplicas   int    `json:"updated_replicas"`
	Image             string `json:"image"`
	Status            string `json:"status"` // HEALTHY, PROGRESSING, DEGRADED
}

type K8sClusterEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // Normal, Warning
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Object    string    `json:"object"`
	Namespace string    `json:"namespace"`
	Timestamp time.Time `json:"timestamp"`
}

type K8sIngestPayload struct {
	ClusterName  string            `json:"cluster_name"`
	Version      string            `json:"version"`
	Nodes        []K8sNode         `json:"nodes,omitempty"`
	Deployments  []K8sDeployment   `json:"deployments,omitempty"`
	Pods         []K8sPod          `json:"pods,omitempty"`
	Events       []K8sClusterEvent `json:"events,omitempty"`
	ReportingNode string           `json:"reporting_node,omitempty"`
}

var (
	k8sMu           sync.RWMutex
	liveOverview    K8sClusterOverview
	liveNodes       []K8sNode
	liveDeployments []K8sDeployment
	livePods        []K8sPod
	liveEvents      []K8sClusterEvent
	liveContainers  []DockerContainer
)

func init() {
	seedClusterData()
}

func seedClusterData() {
	k8sMu.Lock()
	defer k8sMu.Unlock()

	now := time.Now().UTC()

	liveOverview = K8sClusterOverview{
		ClusterName:          "prod-us-east-k8s-cluster",
		Version:              "v1.29.2+eks",
		Status:               "HEALTHY",
		TotalNodes:           8,
		ReadyNodes:           8,
		TotalPods:            64,
		RunningPods:          62,
		PendingPods:          2,
		FailedPods:           0,
		TotalCPUMilli:        32000,
		AllocatedCPUMilli:    18400,
		TotalMemoryBytes:     137438953472, // 128 GB
		AllocatedMemoryBytes: 85899345920,  // 80 GB
		LastSyncAt:           now,
		IsLiveCollector:      false,
	}

	liveNodes = []K8sNode{
		{
			Name:           "ip-10-0-12-42.ec2.internal",
			Role:           "worker,infra",
			Status:         "Ready",
			InternalIP:     "10.0.12.42",
			OSImage:        "Ubuntu 22.04.4 LTS",
			KubeletVersion: "v1.29.2",
			CPUUsagePct:    42.5,
			MemoryUsagePct: 61.2,
			MemoryPressure: false,
			DiskPressure:   false,
			PIDPressure:    false,
			PodCount:       12,
			PodCapacity:    29,
			Labels:         map[string]string{"node.kubernetes.io/instance-type": "m6i.xlarge", "topology.kubernetes.io/zone": "us-east-1a"},
		},
		{
			Name:           "ip-10-0-12-88.ec2.internal",
			Role:           "worker,compute",
			Status:         "Ready",
			InternalIP:     "10.0.12.88",
			OSImage:        "Ubuntu 22.04.4 LTS",
			KubeletVersion: "v1.29.2",
			CPUUsagePct:    68.1,
			MemoryUsagePct: 74.8,
			MemoryPressure: false,
			DiskPressure:   false,
			PIDPressure:    false,
			PodCount:       18,
			PodCapacity:    29,
			Labels:         map[string]string{"node.kubernetes.io/instance-type": "c6i.2xlarge", "topology.kubernetes.io/zone": "us-east-1b"},
		},
		{
			Name:           "ip-10-0-14-19.ec2.internal",
			Role:           "worker,database",
			Status:         "Ready",
			InternalIP:     "10.0.14.19",
			OSImage:        "Ubuntu 22.04.4 LTS",
			KubeletVersion: "v1.29.2",
			CPUUsagePct:    82.4,
			MemoryUsagePct: 88.6,
			MemoryPressure: false,
			DiskPressure:   false,
			PIDPressure:    false,
			PodCount:       8,
			PodCapacity:    29,
			Labels:         map[string]string{"node.kubernetes.io/instance-type": "r6i.2xlarge", "topology.kubernetes.io/zone": "us-east-1a"},
		},
	}

	liveDeployments = []K8sDeployment{
		{
			Name:              "sentrix-api-gateway",
			Namespace:         "production",
			DesiredReplicas:   3,
			AvailableReplicas: 3,
			UpdatedReplicas:   3,
			Image:             "sentrix/api:v1.2.4",
			Status:            "HEALTHY",
		},
		{
			Name:              "auth-service",
			Namespace:         "production",
			DesiredReplicas:   2,
			AvailableReplicas: 2,
			UpdatedReplicas:   2,
			Image:             "sentrix/auth:v1.1.0",
			Status:            "HEALTHY",
		},
		{
			Name:              "payments-processor",
			Namespace:         "production",
			DesiredReplicas:   4,
			AvailableReplicas: 3,
			UpdatedReplicas:   4,
			Image:             "sentrix/payments:v2.8.0",
			Status:            "PROGRESSING",
		},
		{
			Name:              "metrics-indexer",
			Namespace:         "observability",
			DesiredReplicas:   2,
			AvailableReplicas: 2,
			UpdatedReplicas:   2,
			Image:             "sentrix/indexer:v1.0.2",
			Status:            "HEALTHY",
		},
	}

	livePods = []K8sPod{
		{
			Name:          "sentrix-api-gateway-7489f6d74c-8kz2p",
			Namespace:     "production",
			Node:          "ip-10-0-12-42.ec2.internal",
			Phase:         "Running",
			RestartCount:  0,
			OOMKilled:     false,
			IP:            "10.0.12.102",
			Image:         "sentrix/api:v1.2.4",
			CPUUsagePct:   18.2,
			CPULimitMilli: 1000,
			MemUsageMB:    284,
			MemLimitMB:    512,
			CreatedAt:     now.Add(-5 * 24 * time.Hour),
		},
		{
			Name:          "sentrix-api-gateway-7489f6d74c-m4v91",
			Namespace:     "production",
			Node:          "ip-10-0-12-88.ec2.internal",
			Phase:         "Running",
			RestartCount:  0,
			OOMKilled:     false,
			IP:            "10.0.12.144",
			Image:         "sentrix/api:v1.2.4",
			CPUUsagePct:   15.4,
			CPULimitMilli: 1000,
			MemUsageMB:    268,
			MemLimitMB:    512,
			CreatedAt:     now.Add(-5 * 24 * time.Hour),
		},
		{
			Name:          "payments-processor-58d97bc8f-x82pt",
			Namespace:     "production",
			Node:          "ip-10-0-14-19.ec2.internal",
			Phase:         "Running",
			RestartCount:  1,
			OOMKilled:     false,
			IP:            "10.0.14.88",
			Image:         "sentrix/payments:v2.8.0",
			CPUUsagePct:   64.8,
			CPULimitMilli: 2000,
			MemUsageMB:    780,
			MemLimitMB:    1024,
			CreatedAt:     now.Add(-1 * 24 * time.Hour),
		},
		{
			Name:          "payments-processor-58d97bc8f-p941q",
			Namespace:     "production",
			Node:          "ip-10-0-12-88.ec2.internal",
			Phase:         "Pending",
			RestartCount:  0,
			OOMKilled:     false,
			IP:            "",
			Image:         "sentrix/payments:v2.8.0",
			CPUUsagePct:   0.0,
			CPULimitMilli: 2000,
			MemUsageMB:    0,
			MemLimitMB:    1024,
			StatusReason:  "Insufficient memory on node",
			CreatedAt:     now.Add(-12 * time.Minute),
		},
		{
			Name:          "auth-service-6869b59c4b-c72lx",
			Namespace:     "production",
			Node:          "ip-10-0-12-42.ec2.internal",
			Phase:         "Running",
			RestartCount:  0,
			OOMKilled:     false,
			IP:            "10.0.12.155",
			Image:         "sentrix/auth:v1.1.0",
			CPUUsagePct:   8.9,
			CPULimitMilli: 500,
			MemUsageMB:    195,
			MemLimitMB:    512,
			CreatedAt:     now.Add(-18 * 24 * time.Hour),
		},
	}

	liveEvents = []K8sClusterEvent{
		{
			ID:        "evt-01",
			Type:      "Warning",
			Reason:    "FailedScheduling",
			Message:   "0/8 nodes are available: 8 Insufficient memory. Pod payments-processor-58d97bc8f-p941q queued for cluster autoscaling.",
			Object:    "Pod/payments-processor-58d97bc8f-p941q",
			Namespace: "production",
			Timestamp: now.Add(-12 * time.Minute),
		},
		{
			ID:        "evt-02",
			Type:      "Normal",
			Reason:    "Pulled",
			Message:   "Container image sentrix/payments:v2.8.0 already present on machine",
			Object:    "Pod/payments-processor-58d97bc8f-x82pt",
			Namespace: "production",
			Timestamp: now.Add(-1 * 24 * time.Hour),
		},
		{
			ID:        "evt-03",
			Type:      "Normal",
			Reason:    "ScalingReplicaSet",
			Message:   "Scaled up replica set payments-processor-58d97bc8f to 4 from 3",
			Object:    "Deployment/payments-processor",
			Namespace: "production",
			Timestamp: now.Add(-20 * time.Minute),
		},
	}

	liveContainers = []DockerContainer{
		{
			ID:               "cnt-nginx-gateway-01",
			Name:             "sentrix-edge-proxy",
			Image:            "nginx:1.25-alpine",
			Status:           "RUNNING",
			ServerID:         "srv-prod-api-01",
			ServerName:       "api-gw-us-east-1",
			CPUUsagePct:      14.2,
			MemoryUsageBytes: 134217728,  // 128 MB
			MemoryLimitBytes: 1073741824, // 1 GB
			NetRxBytes:       458291000,
			NetTxBytes:       892301900,
			RestartCount:     0,
			OOMKilled:        false,
			CreatedAt:        now.Add(-14 * 24 * time.Hour),
		},
		{
			ID:               "cnt-pg-pooler-02",
			Name:             "pgbouncer-primary",
			Image:            "edoburu/pgbouncer:v1.21.0",
			Status:           "RUNNING",
			ServerID:         "srv-prod-db-primary",
			ServerName:       "db-primary-eu-west-1",
			CPUUsagePct:      48.6,
			MemoryUsageBytes: 314572800,  // 300 MB
			MemoryLimitBytes: 2147483648, // 2 GB
			NetRxBytes:       1289104000,
			NetTxBytes:       1948201000,
			RestartCount:     1,
			OOMKilled:        false,
			CreatedAt:        now.Add(-30 * 24 * time.Hour),
		},
		{
			ID:               "cnt-redis-cache-03",
			Name:             "redis-session-store",
			Image:            "redis:7.2-alpine",
			Status:           "RUNNING",
			ServerID:         "srv-prod-api-02",
			ServerName:       "api-gw-us-east-2",
			CPUUsagePct:      6.8,
			MemoryUsageBytes: 524288000,  // 500 MB
			MemoryLimitBytes: 4294967296, // 4 GB
			NetRxBytes:       682910000,
			NetTxBytes:       912301000,
			RestartCount:     0,
			OOMKilled:        false,
			CreatedAt:        now.Add(-60 * 24 * time.Hour),
		},
		{
			ID:               "cnt-worker-job-04",
			Name:             "async-billing-consumer",
			Image:            "sentrix/billing-worker:v2.4.1",
			Status:           "RUNNING",
			ServerID:         "srv-prod-worker-01",
			ServerName:       "worker-us-east-1",
			CPUUsagePct:      62.1,
			MemoryUsageBytes: 891289600,  // 850 MB
			MemoryLimitBytes: 1073741824, // 1 GB
			NetRxBytes:       142910000,
			NetTxBytes:       189201000,
			RestartCount:     2,
			OOMKilled:        false,
			CreatedAt:        now.Add(-4 * 24 * time.Hour),
		},
		{
			ID:               "cnt-otel-collector-05",
			Name:             "otel-collector-daemon",
			Image:            "otel/opentelemetry-collector-contrib:0.95.0",
			Status:           "RUNNING",
			ServerID:         "srv-prod-api-01",
			ServerName:       "api-gw-us-east-1",
			CPUUsagePct:      4.1,
			MemoryUsageBytes: 188743680, // 180 MB
			MemoryLimitBytes: 1073741824,
			NetRxBytes:       891281000,
			NetTxBytes:       891281000,
			RestartCount:     0,
			OOMKilled:        false,
			CreatedAt:        now.Add(-14 * 24 * time.Hour),
		},
	}
}

// GET /api/v1/containers
func HandleListContainers(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		k8sMu.RLock()
		defer k8sMu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(liveContainers)
	}
}

// GET /api/v1/kubernetes/overview
func HandleGetK8sOverview(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		k8sMu.RLock()
		defer k8sMu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(liveOverview)
	}
}

// GET /api/v1/kubernetes/nodes
func HandleGetK8sNodes(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		k8sMu.RLock()
		defer k8sMu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(liveNodes)
	}
}

// GET /api/v1/kubernetes/workloads
func HandleGetK8sWorkloads(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		k8sMu.RLock()
		defer k8sMu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"deployments": liveDeployments,
			"pods":        livePods,
			"events":      liveEvents,
		})
	}
}

// POST /api/v1/kubernetes/ingest
func HandleIngestK8sMetrics(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload K8sIngestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
			return
		}

		k8sMu.Lock()
		defer k8sMu.Unlock()

		if payload.ClusterName != "" {
			liveOverview.ClusterName = payload.ClusterName
		}
		if payload.Version != "" {
			liveOverview.Version = payload.Version
		}
		liveOverview.IsLiveCollector = true
		liveOverview.LastSyncAt = time.Now().UTC()

		if len(payload.Nodes) > 0 {
			liveNodes = payload.Nodes
			liveOverview.TotalNodes = len(liveNodes)
			ready := 0
			for _, n := range liveNodes {
				if n.Status == "Ready" {
					ready++
				}
			}
			liveOverview.ReadyNodes = ready
		}

		if len(payload.Deployments) > 0 {
			liveDeployments = payload.Deployments
		}

		if len(payload.Pods) > 0 {
			livePods = payload.Pods
			liveOverview.TotalPods = len(livePods)
			running, pending, failed := 0, 0, 0
			var allocatedCPU int64
			var allocatedMem uint64
			for _, p := range livePods {
				switch p.Phase {
				case "Running":
					running++
				case "Pending":
					pending++
				case "Failed", "CrashLoopBackOff":
					failed++
				}
				allocatedCPU += p.CPULimitMilli
				allocatedMem += uint64(p.MemLimitMB * 1024 * 1024)
			}
			liveOverview.RunningPods = running
			liveOverview.PendingPods = pending
			liveOverview.FailedPods = failed
			if allocatedCPU > 0 {
				liveOverview.AllocatedCPUMilli = allocatedCPU
			}
			if allocatedMem > 0 {
				liveOverview.AllocatedMemoryBytes = allocatedMem
			}
		}

		if len(payload.Events) > 0 {
			// Prepend new events
			liveEvents = append(payload.Events, liveEvents...)
			if len(liveEvents) > 100 {
				liveEvents = liveEvents[:100]
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"status":       "accepted",
			"cluster_name": liveOverview.ClusterName,
			"nodes_count":  len(liveNodes),
			"pods_count":   len(livePods),
		})
	}
}

// POST /api/v1/containers/ingest
func HandleIngestContainerMetrics(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var incoming []DockerContainer
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
			return
		}

		k8sMu.Lock()
		defer k8sMu.Unlock()

		if len(incoming) > 0 {
			liveContainers = incoming
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"status":            "accepted",
			"containers_stored": len(incoming),
		})
	}
}

// GET /deploy/sentrix-daemonset.yaml and GET /api/v1/kubernetes/daemonset.yaml
func HandleServeK8sDaemonSet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		manifestPaths := []string{
			"deploy/kubernetes/sentrix-daemonset.yaml",
			"../deploy/kubernetes/sentrix-daemonset.yaml",
			"../../deploy/kubernetes/sentrix-daemonset.yaml",
		}

		for _, p := range manifestPaths {
			if data, err := os.ReadFile(p); err == nil {
				w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
				w.Header().Set("Content-Disposition", "inline; filename=\"sentrix-daemonset.yaml\"")
				w.WriteHeader(http.StatusOK)
				w.Write(data)
				return
			}
		}

		http.Error(w, "DaemonSet manifest template not found", http.StatusNotFound)
	}
}
