package rpc

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessageBuilders(t *testing.T) {
	ts := time.Now().UnixMilli()
	req := NewRequest("1", "File.Get", map[string]any{"id": 7}, ts)
	if req.Header.Type != TypeRequest || req.Header.Method != "File.Get" {
		b, _ := json.Marshal(req)
		if req.Header.Type != TypeRequest {
			t.Fatalf("wrong type: %s json=%s", req.Header.Type, string(b))
		}
	}

	res := NewResponse("1", map[string]any{"name": "ok"}, ts, true)
	if !res.Header.Final || res.Header.Type != TypeResponse {
		t.Fatalf("response final/type mismatch: %+v", res.Header)
	}

	err := NewError("1", "NOT_FOUND", "missing", map[string]string{"detail": "x"}, ts)
	if err.Error == nil || err.Error.Code != "NOT_FOUND" || err.Header.Type != TypeError {
		t.Fatalf("error message mismatch: %+v", err)
	}

	evt := NewEvent("ch:1", 3, false, "payload", ts)
	if evt.Header.Channel != "ch:1" || evt.Header.Type != TypeEvent {
		t.Fatalf("event mismatch: %+v", evt.Header)
	}

	ack := NewAck("1", 3, ts, false)
	if ack.Header.Type != TypeAck || ack.Header.Seq != 3 {
		t.Fatalf("ack mismatch: %+v", ack.Header)
	}

	// JSON encode/decode roundtrip
	data, e := json.Marshal(req)
	if e != nil {
		t.Fatalf("marshal: %v", e)
	}
	var decoded Message
	if e = json.Unmarshal(data, &decoded); e != nil {
		t.Fatalf("unmarshal: %v", e)
	}
	if decoded.Header.ID != req.Header.ID || decoded.Header.Method != req.Header.Method {
		t.Fatalf("roundtrip mismatch: %+v vs %+v", decoded.Header, req.Header)
	}
}
