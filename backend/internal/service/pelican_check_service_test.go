//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIPelicanJudgeSendsMaxEffortAndParsesVerdict(t *testing.T) {
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"` + "```json\\n{\\\"verdict\\\":\\\"degraded\\\",\\\"confidence\\\":\\\"high\\\",\\\"reason\\\":\\\"no bicycle\\\",\\\"signals\\\":[\\\"missing bicycle\\\"]}\\n```" + `"}}]}`))
	}))
	defer server.Close()

	judge := NewOpenAIPelicanJudge(PelicanJudgeConfig{
		BaseURL: server.URL + "/v1",
		APIKey:  "test-key",
	})
	result, err := judge.Judge(context.Background(), PelicanJudgeInput{
		AccountID:   7,
		AccountName: "acct",
		Platform:    PlatformAnthropic,
		Model:       "claude-sonnet-4-6",
		HTML:        "<html>pelican</html>",
	})
	require.NoError(t, err)
	require.Equal(t, PelicanVerdictDegraded, result.Verdict)
	require.Equal(t, PelicanConfidenceHigh, result.Confidence)
	require.Equal(t, "no bicycle", result.Reason)
	require.Equal(t, "deepseek-4.1-flash-1017", payload["model"])
	require.Equal(t, "max", payload["reasoning_effort"])
}

func TestPelicanJudgeRejectsUnknownVerdict(t *testing.T) {
	_, err := parsePelicanJudgeResult(`{"verdict":"bad","confidence":"high","reason":"x"}`)
	require.Error(t, err)
}

func TestPelicanDegradedStreakCountsConsecutiveHighConfidenceOnly(t *testing.T) {
	checks := []*PelicanCheck{
		{Status: pelicanCheckStatusCompleted, Verdict: PelicanVerdictDegraded, Confidence: PelicanConfidenceHigh},
		{Status: pelicanCheckStatusCompleted, Verdict: PelicanVerdictDegraded, Confidence: PelicanConfidenceHigh},
		{Status: pelicanCheckStatusCompleted, Verdict: PelicanVerdictHealthy, Confidence: PelicanConfidenceHigh},
		{Status: pelicanCheckStatusCompleted, Verdict: PelicanVerdictDegraded, Confidence: PelicanConfidenceHigh},
	}
	require.Equal(t, 2, pelicanDegradedStreak(checks))
}

func TestPelicanStreakIgnoresNonCompletedRowsAndStopsAtMediumConfidence(t *testing.T) {
	checks := []*PelicanCheck{
		{Status: pelicanCheckStatusCompleted, Verdict: PelicanVerdictDegraded, Confidence: PelicanConfidenceHigh},
		{Status: pelicanCheckStatusFailed},
		{Status: pelicanCheckStatusCompleted, Verdict: PelicanVerdictDegraded, Confidence: PelicanConfidenceMedium},
	}
	require.Equal(t, 1, pelicanDegradedStreak(checks))
}

func TestPelicanPayloadsCarryPromptAndOutputBudget(t *testing.T) {
	options := AccountTestOptions{Prompt: PelicanQuestion(), MaxOutputTokens: 8192}

	claudePayload, err := createTestPayload("claude-sonnet-4-6", options)
	require.NoError(t, err)
	require.Equal(t, 8192, claudePayload["max_tokens"])
	messages := claudePayload["messages"].([]map[string]any)
	content := messages[0]["content"].([]map[string]any)
	require.Equal(t, PelicanQuestion(), content[0]["text"])

	openAIPayload := createOpenAITestPayload("gpt-5", false, options)
	require.Equal(t, 8192, openAIPayload["max_output_tokens"])
	input := openAIPayload["input"].([]map[string]any)
	inputContent := input[0]["content"].([]map[string]any)
	require.Equal(t, PelicanQuestion(), inputContent[0]["text"])

	geminiPayload := createGeminiTestPayload("gemini-2.5-pro", PelicanQuestion(), 8192)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(geminiPayload, &decoded))
	generationConfig := decoded["generationConfig"].(map[string]any)
	require.Equal(t, float64(8192), generationConfig["maxOutputTokens"])
}