package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// PelicanProbeResult is the raw text generation evidence captured from one
// account. Judging and persistence live in PelicanCheckService so the probe
// stays a pure transport concern.
type PelicanProbeResult struct {
	ResponseText string
	LatencyMs    int64
}

const maxPelicanProbeTextBytes = 256 << 10

// RunPelicanProbe drives the existing per-platform account test paths with a
// custom generation prompt and returns the concatenated streamed text.
func (s *AccountTestService) RunPelicanProbe(ctx context.Context, accountID int64, modelID string, prompt string, maxOutputTokens int) (PelicanProbeResult, error) {
	startedAt := time.Now()
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = (&http.Request{}).WithContext(ctx)
	ginCtx.Set(accountTestBackgroundKey, true)

	testErr := s.TestAccountConnection(ginCtx, accountID, modelID, prompt, AccountTestModeDefault, AccountTestOptions{
		Prompt:          prompt,
		MaxOutputTokens: maxOutputTokens,
	})
	finishedAt := time.Now()
	responseText, errMsg := parseTestSSEOutput(w.Body.String())
	responseText = truncatePelicanProbeText(responseText)

	if testErr != nil {
		if errMsg == "" {
			errMsg = testErr.Error()
		}
		return PelicanProbeResult{ResponseText: responseText, LatencyMs: finishedAt.Sub(startedAt).Milliseconds()}, errors.New(errMsg)
	}
	if errMsg != "" {
		return PelicanProbeResult{ResponseText: responseText, LatencyMs: finishedAt.Sub(startedAt).Milliseconds()}, errors.New(errMsg)
	}
	if strings.TrimSpace(responseText) == "" {
		return PelicanProbeResult{LatencyMs: finishedAt.Sub(startedAt).Milliseconds()}, errors.New("account returned an empty response")
	}
	return PelicanProbeResult{
		ResponseText: responseText,
		LatencyMs:    finishedAt.Sub(startedAt).Milliseconds(),
	}, nil
}

func truncatePelicanProbeText(value string) string {
	if len(value) <= maxPelicanProbeTextBytes {
		return value
	}
	runes := []rune(value)
	limit := maxPelicanProbeTextBytes / 2
	if limit > len(runes) {
		limit = len(runes)
	}
	return string(runes[:limit]) + "\n[truncated before judge]"
}
