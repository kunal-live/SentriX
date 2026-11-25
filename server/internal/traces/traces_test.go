package traces

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestOTLPNanosecondInt64Precision(t *testing.T) {
	// Nanosecond timestamps > 2^53 (9007199254740992) lose precision if decoded as float64.
	// 1795584679859123456 != float64(1795584679859123456) -> 1795584679859123456 becomes ...3456 vs ...3424.
	rawJSON := `{
		"resourceSpans": [{
			"resource": {
				"attributes": [{"key": "service.name", "value": {"stringValue": "precision-svc"}}]
			},
			"scopeSpans": [{
				"spans": [{
					"traceId": "trc_prec_1",
					"spanId": "spn_prec_1",
					"name": "GET /precise",
					"startTimeUnixNano": 1795584679859123456,
					"endTimeUnixNano": 1795584679871456789,
					"attributes": [{"key": "http.status_code", "value": {"intValue": 200}}]
				}]
			}]
		}]
	}`

	var otlp otlpPayload
	dec := json.NewDecoder(bytes.NewReader([]byte(rawJSON)))
	dec.UseNumber()
	if err := dec.Decode(&otlp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	s := otlp.ResourceSpans[0].ScopeSpans[0].Spans[0]
	start := parseNano(s.StartTimeUnixNano)
	end := parseNano(s.EndTimeUnixNano)

	if start != 1795584679859123456 {
		t.Fatalf("lost nanosecond precision on startTimeUnixNano: got %d, want 1795584679859123456", start)
	}
	if end != 1795584679871456789 {
		t.Fatalf("lost nanosecond precision on endTimeUnixNano: got %d, want 1795584679871456789", end)
	}
}
