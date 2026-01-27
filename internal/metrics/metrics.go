// Package metrics provides Prometheus metrics collection for GateKey.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "gatekey"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "http_requests_in_flight",
			Help:      "Number of HTTP requests currently being processed",
		},
	)

	// Authentication metrics
	AuthAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "auth_attempts_total",
			Help:      "Total number of authentication attempts",
		},
		[]string{"provider", "status"},
	)

	AuthSessionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "auth_sessions_active",
			Help:      "Number of active user sessions",
		},
	)

	// Gateway metrics
	GatewaysTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "gateways_total",
			Help:      "Total number of gateways by status",
		},
		[]string{"status", "type"},
	)

	GatewayHeartbeatsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "gateway_heartbeats_total",
			Help:      "Total number of gateway heartbeats received",
		},
		[]string{"gateway_id"},
	)

	GatewayLastHeartbeat = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "gateway_last_heartbeat_timestamp",
			Help:      "Unix timestamp of last heartbeat from gateway",
		},
		[]string{"gateway_id"},
	)

	// VPN connection metrics
	VPNConnectionsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "vpn_connections_active",
			Help:      "Number of active VPN connections",
		},
		[]string{"gateway_id", "type"},
	)

	VPNConnectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "vpn_connections_total",
			Help:      "Total number of VPN connections established",
		},
		[]string{"gateway_id", "type", "status"},
	)

	VPNBytesTransferred = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "vpn_bytes_transferred_total",
			Help:      "Total bytes transferred through VPN",
		},
		[]string{"gateway_id", "direction"},
	)

	// Certificate metrics
	CertificatesIssuedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "certificates_issued_total",
			Help:      "Total number of certificates issued",
		},
		[]string{"type"},
	)

	CertificatesRevokedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "certificates_revoked_total",
			Help:      "Total number of certificates revoked",
		},
	)

	CertificateExpiryDays = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "certificate_expiry_days",
			Help:      "Days until certificate expires",
		},
		[]string{"type", "common_name"},
	)

	// Config generation metrics
	ConfigsGeneratedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "configs_generated_total",
			Help:      "Total number of VPN configurations generated",
		},
		[]string{"type", "gateway_id"},
	)

	// Database metrics
	DBConnectionsOpen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_connections_open",
			Help:      "Number of open database connections",
		},
	)

	DBConnectionsInUse = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_connections_in_use",
			Help:      "Number of database connections in use",
		},
	)

	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "db_query_duration_seconds",
			Help:      "Database query duration in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"operation"},
	)

	// API key metrics
	APIKeyUsageTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "api_key_usage_total",
			Help:      "Total number of API key usages",
		},
		[]string{"key_id", "endpoint"},
	)

	// Policy evaluation metrics
	PolicyEvaluationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "policy_evaluations_total",
			Help:      "Total number of policy evaluations",
		},
		[]string{"result"},
	)

	PolicyEvaluationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "policy_evaluation_duration_seconds",
			Help:      "Policy evaluation duration in seconds",
			Buckets:   []float64{.0001, .0005, .001, .005, .01, .025, .05, .1},
		},
	)

	// Server info metric
	ServerInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "server_info",
			Help:      "GateKey server information",
		},
		[]string{"version", "go_version"},
	)
)

// RecordHTTPRequest records metrics for an HTTP request.
func RecordHTTPRequest(method, path, status string, duration float64) {
	HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
}

// RecordAuthAttempt records an authentication attempt.
func RecordAuthAttempt(provider string, success bool) {
	status := "failure"
	if success {
		status = "success"
	}
	AuthAttemptsTotal.WithLabelValues(provider, status).Inc()
}

// RecordGatewayHeartbeat records a gateway heartbeat.
func RecordGatewayHeartbeat(gatewayID string, timestamp float64) {
	GatewayHeartbeatsTotal.WithLabelValues(gatewayID).Inc()
	GatewayLastHeartbeat.WithLabelValues(gatewayID).Set(timestamp)
}

// RecordVPNConnection records a VPN connection event.
func RecordVPNConnection(gatewayID, connType, status string) {
	VPNConnectionsTotal.WithLabelValues(gatewayID, connType, status).Inc()
}

// RecordCertificateIssued records a certificate issuance.
func RecordCertificateIssued(certType string) {
	CertificatesIssuedTotal.WithLabelValues(certType).Inc()
}

// RecordConfigGenerated records a config generation.
func RecordConfigGenerated(configType, gatewayID string) {
	ConfigsGeneratedTotal.WithLabelValues(configType, gatewayID).Inc()
}

// RecordPolicyEvaluation records a policy evaluation.
func RecordPolicyEvaluation(allowed bool, duration float64) {
	result := "denied"
	if allowed {
		result = "allowed"
	}
	PolicyEvaluationsTotal.WithLabelValues(result).Inc()
	PolicyEvaluationDuration.Observe(duration)
}
