package claude

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func fakeServer(t *testing.T, stopReason, text string, gotBody *map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if gotBody != nil {
			_ = json.Unmarshal(body, gotBody)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_test", "type": "message", "role": "assistant", "model": "claude-opus-5-5",
			"stop_reason": stopReason,
			"content":     []map[string]any{{"type": "text", "text": text}},
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
}

func testClient(url string) *Client {
	c := NewClient(Config{APIKey: "test"})
	c.api = anthropic.NewClient(option.WithAPIKey("test"), option.WithBaseURL(url), option.WithMaxRetries(0))
	return c
}

func TestGenerateScenarios(t *testing.T) {
	resp := `{"scenarios":[{"pain_point":"нет времени","title":"Утро","scenes":[{"position":1,"duration_sec":3,"description":"хук"},{"position":2,"duration_sec":7,"description":"продукт"}]}]}`
	var body map[string]any
	srv := fakeServer(t, "end_turn", resp, &body)
	defer srv.Close()

	got, err := testClient(srv.URL).GenerateScenarios(context.Background(), ScenarioJob{
		ProductDescription: "термокружка", Count: 1, ScenesPerScenario: 2, SceneDurationSec: 5, Language: "русский",
		Images: []ProductImage{{MediaType: "image/png", Data: []byte{1, 2, 3}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "Утро" || len(got[0].Scenes) != 2 || got[0].Scenes[1].DurationSec != 7 {
		t.Fatalf("unexpected result: %+v", got)
	}

	format := body["output_config"].(map[string]any)["format"].(map[string]any)
	if format["type"] != "json_schema" || format["schema"] == nil {
		t.Fatalf("structured output not sent: %v", body["output_config"])
	}
	if body["fallbacks"] != "default" {
		t.Fatalf("fallbacks not sent: %v", body["fallbacks"])
	}
	content := body["messages"].([]any)[0].(map[string]any)["content"].([]any)
	if content[0].(map[string]any)["type"] != "image" {
		t.Fatalf("image block must go first: %v", content[0])
	}
}

func TestGenerateScenariosErrors(t *testing.T) {
	cases := []struct {
		name, stop, text string
		want             error
	}{
		{"refusal", "refusal", "", ErrRefusal},
		{"truncated", "max_tokens", `{"scenarios":[`, ErrTruncated},
		{"empty scenarios", "end_turn", `{"scenarios":[]}`, ErrInvalidResult},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := fakeServer(t, tc.stop, tc.text, nil)
			defer srv.Close()
			_, err := testClient(srv.URL).GenerateScenarios(context.Background(), ScenarioJob{Count: 1})
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}
}
