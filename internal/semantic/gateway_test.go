package semantic_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/archrail/archrail/internal/engine"
	"github.com/archrail/archrail/internal/semantic"
)

const testKey = "vck_SUPER_SECRET_TOKEN_DO_NOT_LEAK"

func TestGatewayParsesVerdict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond with a fenced JSON verdict to exercise extraction.
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant",
					"content": "```json\n{\"conforms\": false, \"findings\": [{\"file\": \"a.ts\", \"message\": \"nope\"}]}\n```"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	gw := semantic.NewGateway(testKey, "anthropic/claude-sonnet-4-6", srv.URL)
	v, err := gw.Evaluate(engine.SemanticRequest{Assertion: "x", Files: map[string]string{"a.ts": "code"}})
	if err != nil {
		t.Fatal(err)
	}
	if v.Conforms || len(v.Findings) != 1 || v.Findings[0].Message != "nope" {
		t.Fatalf("unexpected verdict: %+v", v)
	}
}

func TestGatewayNeverLeaksKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "unauthorized")
	}))
	defer srv.Close()

	gw := semantic.NewGateway(testKey, "", srv.URL)
	_, err := gw.Evaluate(engine.SemanticRequest{Assertion: "x", Files: map[string]string{"a.ts": "code"}})
	if err == nil {
		t.Fatal("expected error on 401")
	}
	// The key travels only in the Authorization header, never in error output.
	if !strings.Contains(gotAuth, testKey) {
		t.Fatalf("key was not sent via Authorization header: %q", gotAuth)
	}
	if strings.Contains(err.Error(), testKey) {
		t.Fatalf("error message leaked the API key: %v", err)
	}
}
