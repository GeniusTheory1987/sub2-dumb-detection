/**
 * Admin Pelican degradation probe API
 */
import { apiClient } from '../client'

export interface PelicanCheck {
  id: number
  account_id: number
  model: string
  status: string
  verdict: string
  confidence: string
  reason: string
  signals: string[]
  output_excerpt: string
  output_html?: string
  error_message: string
  judge_model: string
  latency_ms: number
  started_at: string
  finished_at: string | null
  created_at: string
}

export interface PelicanAccountOverview {
  account_id: number
  name: string
  platform: string
  account_type: string
  status: string
  schedulable: boolean
  supported: boolean
  running: boolean
  degraded_streak: number
  last_check: PelicanCheck | null
  recent_checks: PelicanCheck[]
}

export interface PelicanOverview {
  configured: boolean
  judge_model: string
  interval_minutes: number
  strict_mode: boolean
  degraded_streak_to_pause: number
  accounts: PelicanAccountOverview[]
}

export async function getOverview(): Promise<PelicanOverview> {
  const { data } = await apiClient.get<PelicanOverview>('/admin/pelican')
  return data
}

export async function runAccount(accountId: number, model?: string): Promise<void> {
  await apiClient.post(`/admin/pelican/accounts/${accountId}/run`, model ? { model } : {})
}

export async function listChecks(accountId: number, limit = 20): Promise<PelicanCheck[]> {
  const { data } = await apiClient.get<PelicanCheck[]>(
    `/admin/pelican/accounts/${accountId}/checks`,
    { params: { limit } }
  )
  return data ?? []
}

const pelicanAPI = {
  getOverview,
  runAccount,
  listChecks
}

export default pelicanAPI