package network

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sentrix/server/internal/api"
)

type NetworkInterfaceStats struct {
	Name          string  `json:"name"`
	IP            string  `json:"ip"`
	RxMbps        float64 `json:"rx_mbps"`
	TxMbps        float64 `json:"tx_mbps"`
	RxPacketsSec  float64 `json:"rx_packets_sec"`
	TxPacketsSec  float64 `json:"tx_packets_sec"`
	RxDropPercent float64 `json:"rx_drop_percent"`
	TxDropPercent float64 `json:"tx_drop_percent"`
	Status        string  `json:"status"`
}

type TCPMetrics struct {
	TotalConnections      int     `json:"total_connections"`
	RetransmitsPerSec     float64 `json:"retransmits_per_sec"`
	RetransmitRatePercent float64 `json:"retransmit_rate_percent"`
	ActiveOpensSec        float64 `json:"active_opens_sec"`
	PassiveOpensSec       float64 `json:"passive_opens_sec"`
	ResetCount            int     `json:"reset_count"`
}

type DNSMetrics struct {
	MeanLatencyMs      float64 `json:"mean_latency_ms"`
	P99LatencyMs       float64 `json:"p99_latency_ms"`
	QueriesPerSec      float64 `json:"queries_per_sec"`
	FailureRatePercent float64 `json:"failure_rate_percent"`
	NameserverTarget   string  `json:"nameserver_target"`
}

type TopFlow struct {
	SrcIP         string  `json:"src_ip"`
	DstIP         string  `json:"dst_ip"`
	Port          int     `json:"port"`
	Protocol      string  `json:"protocol"`
	BandwidthMbps float64 `json:"bandwidth_mbps"`
	Service       string  `json:"service"`
}

type NetworkDiagnosticsResponse struct {
	Interfaces   []NetworkInterfaceStats `json:"interfaces"`
	TCPMetrics   TCPMetrics              `json:"tcp_metrics"`
	DNSMetrics   DNSMetrics              `json:"dns_metrics"`
	SocketStates map[string]int          `json:"socket_states"`
	TopFlows     []TopFlow               `json:"top_flows"`
	Status       string                  `json:"status"`
	Timestamp    time.Time               `json:"timestamp"`
}

var defaultDiagnostics = NetworkDiagnosticsResponse{
	Interfaces: []NetworkInterfaceStats{
		{
			Name:          "eth0 (WAN Ingress)",
			IP:            "10.244.0.15/24",
			RxMbps:        18.49,
			TxMbps:        24.12,
			RxPacketsSec:  14200,
			TxPacketsSec:  18650,
			RxDropPercent: 0.00,
			TxDropPercent: 0.00,
			Status:        "UP / OPERATIONAL",
		},
		{
			Name:          "wg0 (WireGuard Mesh)",
			IP:            "10.8.0.1/32",
			RxMbps:        8.15,
			TxMbps:        11.40,
			RxPacketsSec:  6320,
			TxPacketsSec:  8940,
			RxDropPercent: 0.01,
			TxDropPercent: 0.00,
			Status:        "UP / ENCRYPTED",
		},
		{
			Name:          "cni0 (Bridge Mesh)",
			IP:            "172.17.0.1/16",
			RxMbps:        42.30,
			TxMbps:        38.60,
			RxPacketsSec:  31200,
			TxPacketsSec:  29400,
			RxDropPercent: 0.00,
			TxDropPercent: 0.00,
			Status:        "UP / ACTIVE",
		},
		{
			Name:          "lo (Loopback)",
			IP:            "127.0.0.1/8",
			RxMbps:        0.82,
			TxMbps:        0.82,
			RxPacketsSec:  1200,
			TxPacketsSec:  1200,
			RxDropPercent: 0.00,
			TxDropPercent: 0.00,
			Status:        "UP / LOCAL",
		},
	},
	TCPMetrics: TCPMetrics{
		TotalConnections:      4820,
		RetransmitsPerSec:     12.4,
		RetransmitRatePercent: 0.08,
		ActiveOpensSec:        145.2,
		PassiveOpensSec:       210.8,
		ResetCount:            2,
	},
	DNSMetrics: DNSMetrics{
		MeanLatencyMs:      1.20,
		P99LatencyMs:       6.10,
		QueriesPerSec:      840,
		FailureRatePercent: 0.02,
		NameserverTarget:   "10.96.0.10:53 (CoreDNS)",
	},
	SocketStates: map[string]int{
		"ESTABLISHED": 3840,
		"TIME_WAIT":   640,
		"CLOSE_WAIT":  120,
		"LISTEN":      140,
		"SYN_SENT":    45,
		"FIN_WAIT":    35,
	},
	TopFlows: []TopFlow{
		{
			SrcIP:         "10.244.1.12",
			DstIP:         "10.244.2.40",
			Port:          5432,
			Protocol:      "TCP",
			BandwidthMbps: 12.8,
			Service:       "api-gw -> timescale-db",
		},
		{
			SrcIP:         "10.244.1.18",
			DstIP:         "10.244.3.11",
			Port:          6379,
			Protocol:      "TCP",
			BandwidthMbps: 6.4,
			Service:       "auth-service -> redis-cache",
		},
		{
			SrcIP:         "10.244.0.15",
			DstIP:         "10.244.4.88",
			Port:          9092,
			Protocol:      "TCP",
			BandwidthMbps: 19.5,
			Service:       "ingest-agent -> kafka-broker",
		},
		{
			SrcIP:         "10.244.2.14",
			DstIP:         "10.244.5.21",
			Port:          443,
			Protocol:      "TLS/HTTPS",
			BandwidthMbps: 3.2,
			Service:       "billing-worker -> stripe-api",
		},
	},
	Status:    "HEALTHY",
	Timestamp: time.Now(),
}

// GET /api/v1/network/diagnostics
func HandleGetNetworkDiagnostics(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defaultDiagnostics.Timestamp = time.Now()
		api.RespondJSON(w, http.StatusOK, defaultDiagnostics)
	}
}
