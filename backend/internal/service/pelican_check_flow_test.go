//go:build unit

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type pelicanFakeProbe struct {
	mu    sync.Mutex
	calls int
	text  string
}

func (p *pelicanFakeProbe) RunPelicanProbe(_ context.Context, _ int64, _ string, _ string, _ int) (PelicanProbeResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	return PelicanProbeResult{ResponseText: p.text, LatencyMs: 42}, nil
}

type pelicanFakeJudge struct {
	result PelicanJudgeResult
	err    error
}

func (j pelicanFakeJudge) Judge(context.Context, PelicanJudgeInput) (PelicanJudgeResult, error) {
	return j.result, j.err
}
func (j pelicanFakeJudge) Configured() bool { return true }
func (j pelicanFakeJudge) Model() string    { return "deepseek-4.1-flash-1017" }

type pelicanFakeCheckRepo struct {
	rows   []*PelicanCheck
	nextID int64
}

func (r *pelicanFakeCheckRepo) Create(_ context.Context, check *PelicanCheck) error {
	r.nextID++
	check.ID = r.nextID
	check.CreatedAt = time.Now().Add(time.Duration(r.nextID) * time.Millisecond).UTC()
	r.rows = append(r.rows, check)
	return nil
}

func (r *pelicanFakeCheckRepo) Finish(_ context.Context, check *PelicanCheck) error {
	for _, row := range r.rows {
		if row.ID == check.ID {
			*row = *check
			return nil
		}
	}
	return nil
}

func (r *pelicanFakeCheckRepo) ListRecentByAccount(context.Context, int64, int) ([]*PelicanCheck, error) {
	out := make([]*PelicanCheck, 0, len(r.rows))
	for i := len(r.rows) - 1; i >= 0; i-- {
		out = append(out, r.rows[i])
	}
	return out, nil
}

func (r *pelicanFakeCheckRepo) ListRecentByAccountWithHTML(ctx context.Context, accountID int64, limit int) ([]*PelicanCheck, error) {
	return r.ListRecentByAccount(ctx, accountID, limit)
}

type pelicanFakeAccounts struct {
	AccountRepository
	account Account
	others  []Account
	paused  bool
}

func (r *pelicanFakeAccounts) GetByID(context.Context, int64) (*Account, error) {
	account := r.account
	return &account, nil
}

func (r *pelicanFakeAccounts) ListSchedulable(context.Context) ([]Account, error) {
	accounts := []Account{r.account}
	accounts = append(accounts, r.others...)
	return accounts, nil
}

func (r *pelicanFakeAccounts) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return r.ListSchedulable(context.Background())
}

func (r *pelicanFakeAccounts) SetSchedulable(_ context.Context, _ int64, schedulable bool) error {
	r.paused = !schedulable
	return nil
}

type pelicanFakeSettings struct {
	SettingRepository
	strictMode string
}

func (r pelicanFakeSettings) GetValue(_ context.Context, key string) (string, error) {
	if key == SettingKeyPelicanStrictMode && r.strictMode != "" {
		return r.strictMode, nil
	}
	return "", ErrSettingNotFound
}

func newPelicanFlowService(accounts *pelicanFakeAccounts, probe *pelicanFakeProbe, judge pelicanFakeJudge, strictMode string) (*PelicanCheckService, *pelicanFakeCheckRepo) {
	repo := &pelicanFakeCheckRepo{}
	settings := &SettingService{settingRepo: pelicanFakeSettings{strictMode: strictMode}}
	svc := NewPelicanCheckService(accounts, probe, repo, settings, &config.Config{
		Pelican: config.PelicanConfig{
			Enabled:               true,
			TargetModel:           "claude-sonnet-4-6",
			MaxOutputTokens:       8192,
			DegradedStreakToPause: 2,
			JudgeModel:            "deepseek-4.1-flash-1017",
		},
	})
	svc.judge = judge
	return svc, repo
}

func TestPelicanTwoHighConfidenceDegradedChecksRespectStrictMode(t *testing.T) {
	for _, tc := range []struct {
		name       string
		strictMode string
		wantPaused bool
	}{
		{name: "default"},
		{name: "off", strictMode: "false"},
		{name: "on", strictMode: "true", wantPaused: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			accounts := &pelicanFakeAccounts{
				account: Account{ID: 1, Name: "acct-1", Platform: PlatformAnthropic, Type: AccountTypeOAuth, Status: "active", Schedulable: true},
				others:  []Account{{ID: 2, Name: "acct-2", Platform: PlatformAnthropic, Type: AccountTypeOAuth, Status: "active", Schedulable: true}},
			}
			probe := &pelicanFakeProbe{text: "<html>pelican on a bicycle</html>"}
			judge := pelicanFakeJudge{result: PelicanJudgeResult{Verdict: PelicanVerdictDegraded, Confidence: PelicanConfidenceHigh, Reason: "no bicycle"}}
			svc, repo := newPelicanFlowService(accounts, probe, judge, tc.strictMode)
			ctx := context.Background()

			first, err := svc.CheckAccount(ctx, 1, "")
			require.NoError(t, err)
			require.Equal(t, PelicanVerdictDegraded, first.Verdict)
			require.False(t, accounts.paused, "one verdict must not pause the account")

			second, err := svc.CheckAccount(ctx, 1, "")
			require.NoError(t, err)
			require.Equal(t, PelicanVerdictDegraded, second.Verdict)
			require.Len(t, repo.rows, 2, "both verdicts must be recorded in either mode")
			require.Equal(t, tc.wantPaused, accounts.paused)
		})
	}
}

func TestPelicanHealthyVerdictResetsStreak(t *testing.T) {
	accounts := &pelicanFakeAccounts{
		account: Account{ID: 1, Name: "acct-1", Platform: PlatformAnthropic, Type: AccountTypeOAuth, Status: "active", Schedulable: true},
		others:  []Account{{ID: 2, Name: "acct-2", Platform: PlatformAnthropic, Type: AccountTypeOAuth, Status: "active", Schedulable: true}},
	}
	probe := &pelicanFakeProbe{text: "<html>pelican</html>"}
	degraded := pelicanFakeJudge{result: PelicanJudgeResult{Verdict: PelicanVerdictDegraded, Confidence: PelicanConfidenceHigh}}
	svc, _ := newPelicanFlowService(accounts, probe, degraded, "true")
	_, err := svc.CheckAccount(context.Background(), 1, "")
	require.NoError(t, err)

	svc.judge = pelicanFakeJudge{result: PelicanJudgeResult{Verdict: PelicanVerdictHealthy, Confidence: PelicanConfidenceHigh}}
	_, err = svc.CheckAccount(context.Background(), 1, "")
	require.NoError(t, err)
	svc.judge = degraded
	_, err = svc.CheckAccount(context.Background(), 1, "")
	require.NoError(t, err)
	require.False(t, accounts.paused)
}

func TestPelicanDoesNotPauseLastSchedulableAccount(t *testing.T) {
	accounts := &pelicanFakeAccounts{
		account: Account{ID: 1, Name: "only", Platform: PlatformAnthropic, Type: AccountTypeOAuth, Status: "active", Schedulable: true},
	}
	probe := &pelicanFakeProbe{text: "<html>pelican</html>"}
	svc, _ := newPelicanFlowService(accounts, probe, pelicanFakeJudge{result: PelicanJudgeResult{Verdict: PelicanVerdictDegraded, Confidence: PelicanConfidenceHigh}}, "true")
	for range 2 {
		_, err := svc.CheckAccount(context.Background(), 1, "")
		require.NoError(t, err)
	}
	require.False(t, accounts.paused, "never remove the last schedulable account")
}
