package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOnlineReturnsPlayersFromPresenceEnvelope(t *testing.T) {
	tests := []struct {
		name, channel string
		want          int
	}{{"lobby", "arena", 2}, {"empty", "quiet", 0}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v1/realtime/presence/get/"+tc.channel {
					t.Fatalf("request = %s %s", r.Method, r.URL.Path)
				}
				users := []Player{}
				if tc.want == 2 {
					users = []Player{{ID: "p1", DisplayName: "Ada"}, {ID: "p2", DisplayName: "Lin"}}
				}
				json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": map[string]any{"users": users}, "error": nil, "metadata": map[string]any{}})
			}))
			defer srv.Close()
			client := &InfraiClient{BaseURL: srv.URL, APIKey: "test", HTTP: srv.Client()}
			got, err := client.Online(tc.channel)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Users) != tc.want {
				t.Fatalf("users = %d, want %d", len(got.Users), tc.want)
			}
		})
	}
}
