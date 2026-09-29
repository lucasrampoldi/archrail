package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/archrail/archrail/internal/engine"
)

// EnvAPIKey is the only place Archrail reads the gateway credential from.
const EnvAPIKey = "AI_GATEWAY_API_KEY"

// DefaultModel is the Claude model used when none is configured.
const DefaultModel = "anthropic/claude-sonnet-4-6"

// DefaultBaseURL is the Vercel AI Gateway OpenAI-compatible endpoint.
const DefaultBaseURL = "https://ai-gateway.vercel.sh/v1"

// Gateway is a SemanticBackend backed by the Vercel AI Gateway (OpenAI-compatible
// chat completions), using a Claude model.
type Gateway struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewGateway builds a gateway backend. apiKey must be sourced from the
// environment by the caller and is never persisted.
func NewGateway(apiKey, model, baseURL string) *Gateway {
	if model == "" {
		model = DefaultModel
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Gateway{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Model returns the configured model identifier.
func (g *Gateway) Model() string { return g.model }

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

type verdictJSON struct {
	Conforms bool `json:"conforms"`
	Findings []struct {
		File    string `json:"file"`
		Message string `json:"message"`
	} `json:"findings"`
}

// Evaluate sends the assertion + curated context to the model and parses a
// structured verdict.
func (g *Gateway) Evaluate(req engine.SemanticRequest) (engine.SemanticVerdict, error) {
	prompt := buildPrompt(req)
	body, err := json.Marshal(chatRequest{
		Model:       g.model,
		Temperature: 0,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return engine.SemanticVerdict{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return engine.SemanticVerdict{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return engine.SemanticVerdict{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Do not echo the request (which could include the key via headers).
		return engine.SemanticVerdict{}, fmt.Errorf("gateway returned status %d", resp.StatusCode)
	}

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return engine.SemanticVerdict{}, err
	}
	if len(cr.Choices) == 0 {
		return engine.SemanticVerdict{}, fmt.Errorf("gateway returned no choices")
	}
	return parseVerdict(cr.Choices[0].Message.Content)
}

const systemPrompt = `You are an architecture conformance reviewer. You are given a deterministic architecture fingerprint (verified structural facts), a natural-language assertion, and the relevant first-party source files. Decide whether the code satisfies the assertion. Respond with ONLY a JSON object of the form {"conforms": bool, "findings": [{"file": string, "message": string}]}. Do not include prose outside the JSON.`

func buildPrompt(req engine.SemanticRequest) string {
	var b strings.Builder
	b.WriteString("ARCHITECTURE FINGERPRINT:\n")
	b.WriteString(req.Fingerprint)
	b.WriteString("\nASSERTION:\n")
	b.WriteString(req.Assertion)
	b.WriteString("\n\nFILES:\n")
	paths := make([]string, 0, len(req.Files))
	for p := range req.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		fmt.Fprintf(&b, "\n--- %s ---\n%s\n", p, req.Files[p])
	}
	return b.String()
}

func parseVerdict(content string) (engine.SemanticVerdict, error) {
	raw := extractJSON(content)
	var vj verdictJSON
	if err := json.Unmarshal([]byte(raw), &vj); err != nil {
		return engine.SemanticVerdict{}, fmt.Errorf("could not parse verdict JSON: %w", err)
	}
	v := engine.SemanticVerdict{Conforms: vj.Conforms}
	for _, f := range vj.Findings {
		v.Findings = append(v.Findings, engine.SemanticFinding{File: f.File, Message: f.Message})
	}
	return v, nil
}

// extractJSON pulls the first {...} object out of a possibly fenced response.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
