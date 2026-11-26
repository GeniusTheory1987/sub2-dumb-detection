-- Pelican HTML degradation benchmark evidence.
-- Append-only: streaks and quarantine decisions are derived from ordered rows.
CREATE TABLE IF NOT EXISTS pelican_checks (
    id              BIGSERIAL PRIMARY KEY,
    account_id      BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    model           VARCHAR(100) NOT NULL DEFAULT '',
    status          VARCHAR(20) NOT NULL DEFAULT 'running',
    verdict         VARCHAR(20) NOT NULL DEFAULT '',
    confidence      VARCHAR(20) NOT NULL DEFAULT '',
    reason          TEXT NOT NULL DEFAULT '',
    signals         TEXT NOT NULL DEFAULT '[]',
    output_excerpt  TEXT NOT NULL DEFAULT '',
    error_message   TEXT NOT NULL DEFAULT '',
    judge_model     VARCHAR(100) NOT NULL DEFAULT '',
    latency_ms      BIGINT NOT NULL DEFAULT 0,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pelican_checks_account_created
    ON pelican_checks(account_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_pelican_checks_created
    ON pelican_checks(created_at DESC);

COMMENT ON TABLE pelican_checks IS
    'Per-account pelican HTML benchmark results; consecutive high-confidence degraded verdicts pause the account';