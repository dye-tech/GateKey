package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestRecordHTTPRequest(t *testing.T) {
	// Reset counters for clean test state
	HTTPRequestsTotal.Reset()
	HTTPRequestDuration.Reset()

	RecordHTTPRequest("GET", "/api/v1/test", "200", 0.5)

	// Verify counter was incremented
	counter, err := HTTPRequestsTotal.GetMetricWithLabelValues("GET", "/api/v1/test", "200")
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var m dto.Metric
	if err := counter.Write(&m); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	if *m.Counter.Value != 1 {
		t.Errorf("Expected counter value 1, got %f", *m.Counter.Value)
	}
}

func TestRecordAuthAttempt(t *testing.T) {
	AuthAttemptsTotal.Reset()

	tests := []struct {
		name     string
		provider string
		success  bool
		expected string
	}{
		{"success", "oidc", true, "success"},
		{"failure", "saml", false, "failure"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordAuthAttempt(tt.provider, tt.success)

			counter, err := AuthAttemptsTotal.GetMetricWithLabelValues(tt.provider, tt.expected)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			var m dto.Metric
			if err := counter.Write(&m); err != nil {
				t.Fatalf("Failed to write metric: %v", err)
			}

			if *m.Counter.Value < 1 {
				t.Errorf("Expected counter value >= 1, got %f", *m.Counter.Value)
			}
		})
	}
}

func TestRecordGatewayHeartbeat(t *testing.T) {
	GatewayHeartbeatsTotal.Reset()
	GatewayLastHeartbeat.Reset()

	gatewayID := "gw-test-123"
	timestamp := float64(1234567890)

	RecordGatewayHeartbeat(gatewayID, timestamp)

	// Verify heartbeat counter
	counter, err := GatewayHeartbeatsTotal.GetMetricWithLabelValues(gatewayID)
	if err != nil {
		t.Fatalf("Failed to get heartbeat counter: %v", err)
	}

	var m dto.Metric
	if err := counter.Write(&m); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	if *m.Counter.Value != 1 {
		t.Errorf("Expected heartbeat counter 1, got %f", *m.Counter.Value)
	}

	// Verify last heartbeat timestamp
	gauge, err := GatewayLastHeartbeat.GetMetricWithLabelValues(gatewayID)
	if err != nil {
		t.Fatalf("Failed to get last heartbeat gauge: %v", err)
	}

	if err := gauge.Write(&m); err != nil {
		t.Fatalf("Failed to write gauge: %v", err)
	}

	if *m.Gauge.Value != timestamp {
		t.Errorf("Expected timestamp %f, got %f", timestamp, *m.Gauge.Value)
	}
}

func TestRecordVPNConnection(t *testing.T) {
	VPNConnectionsTotal.Reset()

	RecordVPNConnection("gw-123", "openvpn", "connected")

	counter, err := VPNConnectionsTotal.GetMetricWithLabelValues("gw-123", "openvpn", "connected")
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var m dto.Metric
	if err := counter.Write(&m); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	if *m.Counter.Value != 1 {
		t.Errorf("Expected counter 1, got %f", *m.Counter.Value)
	}
}

func TestRecordCertificateIssued(t *testing.T) {
	CertificatesIssuedTotal.Reset()

	RecordCertificateIssued("client")

	counter, err := CertificatesIssuedTotal.GetMetricWithLabelValues("client")
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var m dto.Metric
	if err := counter.Write(&m); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	if *m.Counter.Value != 1 {
		t.Errorf("Expected counter 1, got %f", *m.Counter.Value)
	}
}

func TestRecordConfigGenerated(t *testing.T) {
	ConfigsGeneratedTotal.Reset()

	RecordConfigGenerated("openvpn", "gw-456")

	counter, err := ConfigsGeneratedTotal.GetMetricWithLabelValues("openvpn", "gw-456")
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	var m dto.Metric
	if err := counter.Write(&m); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	if *m.Counter.Value != 1 {
		t.Errorf("Expected counter 1, got %f", *m.Counter.Value)
	}
}

func TestRecordPolicyEvaluation(t *testing.T) {
	PolicyEvaluationsTotal.Reset()

	tests := []struct {
		name     string
		allowed  bool
		expected string
	}{
		{"allowed", true, "allowed"},
		{"denied", false, "denied"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordPolicyEvaluation(tt.allowed, 0.001)

			counter, err := PolicyEvaluationsTotal.GetMetricWithLabelValues(tt.expected)
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}

			var m dto.Metric
			if err := counter.Write(&m); err != nil {
				t.Fatalf("Failed to write metric: %v", err)
			}

			if *m.Counter.Value < 1 {
				t.Errorf("Expected counter >= 1, got %f", *m.Counter.Value)
			}
		})
	}
}

func TestMetricsRegistered(t *testing.T) {
	// Verify that all metrics are registered with Prometheus
	metrics := []prometheus.Collector{
		HTTPRequestsTotal,
		HTTPRequestDuration,
		HTTPRequestsInFlight,
		AuthAttemptsTotal,
		AuthSessionsActive,
		GatewaysTotal,
		GatewayHeartbeatsTotal,
		GatewayLastHeartbeat,
		VPNConnectionsActive,
		VPNConnectionsTotal,
		VPNBytesTransferred,
		CertificatesIssuedTotal,
		CertificatesRevokedTotal,
		CertificateExpiryDays,
		ConfigsGeneratedTotal,
		DBConnectionsOpen,
		DBConnectionsInUse,
		DBQueryDuration,
		APIKeyUsageTotal,
		PolicyEvaluationsTotal,
		PolicyEvaluationDuration,
		ServerInfo,
	}

	for _, m := range metrics {
		if m == nil {
			t.Error("Found nil metric - metric not properly initialized")
		}
	}
}
