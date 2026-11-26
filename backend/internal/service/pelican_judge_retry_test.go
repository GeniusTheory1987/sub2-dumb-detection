package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newPelicanJudgeForTest(t *testing.T, handler http.Handler) (*OpenAIPelicanJudge, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewOpenAIPelicanJudge(PelicanJudgeConfig{
		BaseURL:         server.URL,
		APIKey:          "test-key",
		Model:           "test-model",
		ReasoningEffort: "max",
		Timeout:         5 * time.Second,
		HTTPClient:      server.Client(),
	}), server
}

func writeValidPelicanJudgeResponse(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"content": `{"verdict":"healthy","confidence":"high","reason":"ok","signals":[]}`,
				},
			},
		},
	}))
}

func TestOpenAIPelicanJudgeRetriesTransientServerError(t *testing.T) {
	var calls atomic.Int32
	judge, _ := newPelicanJudgeForTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "upstream overloaded", http.StatusInternalServerError)
			return
		}
		writeValidPelicanJudgeResponse(t, w)
	}))

	result, err := judge.Judge(context.Background(), PelicanJudgeInput{
		AccountID: 1,
		Model:     "test-model",
		HTML:      "<html><body>ok</body></html>",
	})
	require.NoError(t, err)
	require.Equal(t, PelicanVerdictHealthy, result.Verdict)
	require.Equal(t, PelicanConfidenceHigh, result.Confidence)
	require.Equal(t, int32(2), calls.Load())
}

func TestOpenAIPelicanJudgeRetriesEmptyContent(t *testing.T) {
	var calls atomic.Int32
	judge, _ := newPelicanJudgeForTest(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"choices": []any{}}))
			return
		}
		writeValidPelicanJudgeResponse(t, w)
	}))

	result, err := judge.Judge(context.Background(), PelicanJudgeInput{
		AccountID: 1,
		Model:     "test-model",
		HTML:      "<html><body>ok</body></html>",
	})
	require.NoError(t, err)
	require.Equal(t, PelicanVerdictHealthy, result.Verdict)
	require.Equal(t, PelicanConfidenceHigh, result.Confidence)
	require.Equal(t, int32(2), calls.Load())
}
