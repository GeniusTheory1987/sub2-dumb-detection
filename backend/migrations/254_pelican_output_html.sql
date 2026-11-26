-- Full generated HTML for the admin preview. Kept separate from the truncated
-- output_excerpt so list/overview payloads stay small; only the per-account
-- checks endpoint returns it.
ALTER TABLE pelican_checks
    ADD COLUMN IF NOT EXISTS output_html TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN pelican_checks.output_html IS
    'Full (size-capped) generated HTML; returned only by the per-account checks endpoint for preview';
