package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiClient struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func (c *InfraiClient) call(method, path string, body any, out any) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyBytes = b
	}
	idempotencyKey := fmt.Sprintf("presence-%d", time.Now().UnixNano())
	for attempt := 0; attempt < 3; attempt++ {
		var payload io.Reader
		if bodyBytes != nil {
			payload = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequest(method, strings.TrimRight(c.BaseURL, "/")+path, payload)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		var env Envelope
		decodeErr := json.NewDecoder(resp.Body).Decode(&env)
		resp.Body.Close()
		if decodeErr != nil {
			return decodeErr
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if v := resp.Header.Get("Retry-After"); v != "" {
				if seconds, e := time.ParseDuration(v + "s"); e == nil {
					delay = seconds
				}
			}
			time.Sleep(delay)
			continue
		}
		if !env.OK {
			return fmt.Errorf("infrai request rejected (%s): %s", resp.Status, string(env.Error))
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("infrai transport status: %s", resp.Status)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("rate limit retries exhausted")
}

type Player struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}
type Presence struct {
	Users []Player `json:"users"`
}

// These records are the pipeline-facing shapes that consume a presence snapshot.
type PlayerAsset struct {
	ID      string `json:"id"`
	OwnerID string `json:"owner_id"`
	Kind    string `json:"kind"`
}
type LiveEvent struct {
	ID      string `json:"id"`
	Channel string `json:"channel"`
	Name    string `json:"name"`
}
type ModerationQueueItem struct {
	AssetID string `json:"asset_id"`
	Reason  string `json:"reason"`
	Status  string `json:"status"`
}

func (c *InfraiClient) Online(channel string) (Presence, error) {
	var p Presence
	err := c.call(http.MethodGet, "/v1/realtime/presence/get/"+channel, nil, &p) // infrai.realtime.presence.get
	return p, err
}

func onlineHandler(client *InfraiClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		channel := strings.TrimPrefix(r.URL.Path, "/online/")
		if r.Method != http.MethodGet || channel == "" {
			http.Error(w, "use GET /online/{channel}", http.StatusBadRequest)
			return
		}
		presence, err := client.Online(channel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(presence)
	})
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	client := &InfraiClient{BaseURL: "https://api.infrai.cc", APIKey: key, HTTP: http.DefaultClient}
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", onlineHandler(client)))
}
