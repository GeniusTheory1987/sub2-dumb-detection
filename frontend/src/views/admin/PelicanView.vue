<template>
  <AppLayout>
    <div class="w-full min-w-0 space-y-6 pb-8">
      <header
        class="page-header mb-0 rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700 sm:p-6"
      >
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 class="page-title flex items-center gap-2 text-xl font-black text-gray-900 dark:text-white">
              <span class="inline-flex h-8 w-8 items-center justify-center rounded-xl bg-blue-50 text-blue-500 dark:bg-blue-900/30 dark:text-blue-400">
                <Icon name="chart" size="sm" />
              </span>
              {{ t('admin.pelican.title') }}
            </h1>
            <p class="page-description mt-1.5 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.pelican.description', { minutes: overview?.interval_minutes ?? 15 }) }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="reload">
            {{ t('admin.pelican.refresh') }}
          </button>
        </div>

        <div class="mt-4 flex flex-wrap items-center gap-2 border-t border-gray-100 pt-4 dark:border-dark-700">
          <span
            class="inline-flex items-center rounded-md px-2 py-1 text-xs font-medium"
            :class="overview?.configured
              ? 'bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-300'
              : 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'"
          >
            {{ overview?.configured ? t('admin.pelican.judgeConfigured') : t('admin.pelican.judgeMissing') }}
          </span>
          <span v-if="overview?.judge_model" class="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200">
            {{ t('admin.pelican.judgeModel') }}: {{ overview.judge_model }}
          </span>
          <span v-if="overview" class="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200">
            {{ t('admin.pelican.interval', { minutes: overview.interval_minutes }) }}
          </span>
          <span v-if="overview" class="inline-flex items-center rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200">
            {{ overview.strict_mode
              ? t('admin.pelican.threshold', { count: overview.degraded_streak_to_pause })
              : t('admin.pelican.autoPauseOff') }}
          </span>
        </div>
      </header>

      <div
        v-if="error"
        class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300"
      >
        {{ error }}
      </div>

      <div
        v-if="overview && !overview.configured"
        class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-900/20 dark:text-amber-200"
      >
        {{ t('admin.pelican.judgeHint') }}
      </div>

      <DataTable :columns="columns" :data="accounts" :loading="loading">
        <template #cell-account="{ row }">
          <div class="min-w-0">
            <div class="truncate font-medium text-gray-900 dark:text-white">{{ row.name || `#${row.account_id}` }}</div>
            <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
              {{ row.platform }} · {{ row.account_type }}
              <span v-if="!row.supported"> · {{ t('admin.pelican.unsupported') }}</span>
            </div>
          </div>
        </template>

        <template #cell-state="{ row }">
          <div class="flex flex-wrap items-center gap-1.5">
            <span
              class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
              :class="row.schedulable
                ? 'bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-300'
                : 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300'"
            >
              {{ row.schedulable ? t('admin.pelican.schedulable') : t('admin.pelican.paused') }}
            </span>
            <span v-if="row.running" class="inline-flex items-center rounded-md bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-300">
              {{ t('admin.pelican.running') }}
            </span>
          </div>
        </template>

        <template #cell-last="{ row }">
          <div v-if="row.last_check" class="min-w-0">
            <div class="flex flex-wrap items-center gap-1.5">
              <span class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium" :class="verdictClass(row.last_check.verdict, row.last_check.status)">
                {{ verdictLabel(row.last_check.verdict, row.last_check.status) }}
              </span>
              <span class="text-xs text-gray-500 dark:text-dark-400">
                {{ checkSummary(row.last_check) }}
              </span>
            </div>
            <HelpTooltip
              v-if="checkMessage(row.last_check)"
              :content="checkMessage(row.last_check)"
              :delay-ms="2000"
              width-class="w-[32rem] max-w-[calc(100vw-2rem)]"
            >
              <template #trigger>
                <p
                  class="mt-1 line-clamp-2 max-w-xl cursor-help text-xs"
                  :class="row.last_check.status === 'failed' ? 'text-red-600 dark:text-red-300' : 'text-gray-500 dark:text-dark-400'"
                >
                  {{ checkMessage(row.last_check) }}
                </p>
              </template>
            </HelpTooltip>
          </div>
          <span v-else class="text-sm text-gray-400 dark:text-dark-500">{{ t('admin.pelican.noRuns') }}</span>
        </template>

        <template #cell-streak="{ row }">
          <span
            class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
            :class="row.degraded_streak > 0
              ? 'bg-orange-50 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300'
              : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'"
          >
            {{ overview?.strict_mode ? `${row.degraded_streak}/${overview.degraded_streak_to_pause}` : row.degraded_streak }}
          </span>
        </template>

        <template #cell-actions="{ row }">
          <div class="flex flex-wrap items-center gap-2">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="!row.last_check"
              @click="openDetail(row)"
            >
              {{ t('admin.pelican.detail') }}
            </button>
            <button
              type="button"
              class="btn btn-primary"
              :disabled="row.running || !row.supported"
              @click="runAccount(row)"
            >
              {{ t('admin.pelican.runNow') }}
            </button>
          </div>
        </template>

        <template #empty>
          <EmptyState :title="t('admin.pelican.noRuns')" :description="t('admin.pelican.emptyHint')" />
        </template>
      </DataTable>

      <BaseDialog
        :show="showDetail"
        :title="t('admin.pelican.detailTitle')"
        width="full"
        @close="closeDetail"
      >
        <div class="space-y-5">
          <div
            v-if="detailError"
            class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300"
          >
            {{ detailError }}
          </div>

          <div v-if="detailLoading" class="py-10 text-center text-sm text-gray-500 dark:text-dark-400">
            {{ t('admin.pelican.loadingDetail') }}
          </div>

          <template v-else-if="detailCheck">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="truncate text-base font-semibold text-gray-900 dark:text-white">
                  {{ detailRow?.name || `#${detailRow?.account_id}` }}
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                  {{ detailRow?.platform }} · {{ detailRow?.account_type }}
                </div>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <span
                  class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
                  :class="verdictClass(detailCheck.verdict, detailCheck.status)"
                >
                  {{ verdictLabel(detailCheck.verdict, detailCheck.status) }}
                </span>
                <span class="text-xs text-gray-500 dark:text-dark-400">
                  {{ detailCheck.confidence || '-' }}
                </span>
              </div>
            </div>

            <div>
              <div class="flex flex-wrap items-center justify-between gap-2">
                <h4 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.pelican.preview') }}
                </h4>
                <span
                  v-if="previewIsExcerpt"
                  class="text-xs text-amber-600 dark:text-amber-300"
                >
                  {{ t('admin.pelican.previewExcerptHint') }}
                </span>
              </div>
              <div v-if="previewHtml" class="mt-2 space-y-2">
                <div class="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-dark-700">
                  <iframe
                    :srcdoc="previewSrcdoc"
                    sandbox="allow-scripts"
                    referrerpolicy="no-referrer"
                    class="h-[60vh] min-h-[360px] w-full bg-white"
                    title="pelican-preview"
                  ></iframe>
                </div>
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('admin.pelican.previewScriptHint') }}
                </p>
              </div>
              <div
                v-else
                class="mt-2 rounded-xl border border-gray-200 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400"
              >
                {{ t('admin.pelican.previewUnavailable') }}
              </div>
            </div>

            <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              <div class="rounded-xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
                  {{ t('admin.pelican.generatedAt') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatTime(detailCheck.created_at) }}
                </div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
                  {{ t('admin.pelican.modelLabel') }}
                </div>
                <div class="mt-1 truncate text-sm font-medium text-gray-900 dark:text-white">
                  {{ detailCheck.model || '-' }}
                </div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
                  {{ t('admin.pelican.duration') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ formatLatency(detailCheck.latency_ms) }}
                </div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
                  {{ t('admin.pelican.state') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ detailCheck.status || '-' }}
                </div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
                  {{ t('admin.pelican.confidenceLabel') }}
                </div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
                  {{ detailCheck.confidence || '-' }}
                </div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800/60">
                <div class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
                  {{ t('admin.pelican.judgeModel') }}
                </div>
                <div class="mt-1 truncate text-sm font-medium text-gray-900 dark:text-white">
                  {{ detailCheck.judge_model || '-' }}
                </div>
              </div>
            </div>

            <div v-if="detailCheck.reason" class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
              <div class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
                {{ t('admin.pelican.reasonLabel') }}
              </div>
              <p class="mt-1 whitespace-pre-wrap text-sm text-gray-700 dark:text-gray-200">
                {{ detailCheck.reason }}
              </p>
            </div>

            <div v-if="detailCheck.signals?.length" class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
              <div class="text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">
                {{ t('admin.pelican.signalsLabel') }}
              </div>
              <div class="mt-2 flex flex-wrap gap-1.5">
                <span
                  v-for="signal in detailCheck.signals"
                  :key="signal"
                  class="inline-flex items-center rounded-md bg-gray-100 px-2 py-0.5 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200"
                >
                  {{ signal }}
                </span>
              </div>
            </div>

            <div
              v-if="detailCheck.error_message"
              class="rounded-xl border border-red-200 bg-red-50 p-3 dark:border-red-900/60 dark:bg-red-900/20"
            >
              <div class="text-[11px] font-medium uppercase tracking-wide text-red-600 dark:text-red-300">
                {{ t('admin.pelican.errorLabel') }}
              </div>
              <pre class="mt-1 whitespace-pre-wrap break-words text-xs text-red-700 dark:text-red-300">{{ detailCheck.error_message }}</pre>
            </div>

          </template>
        </div>

        <template #footer>
          <button type="button" class="btn btn-secondary" @click="closeDetail">
            {{ t('admin.pelican.close') }}
          </button>
        </template>
      </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { pelicanAPI } from '@/api/admin'
import type { PelicanAccountOverview, PelicanCheck, PelicanOverview } from '@/api/admin/pelican'
import { extractApiErrorMessage } from '@/utils/apiError'
import { buildPreviewDocument } from '@/utils/htmlPreview'

const { t } = useI18n()

const overview = ref<PelicanOverview | null>(null)
const loading = ref(false)
const error = ref('')
const showDetail = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const detailRow = ref<PelicanAccountOverview | null>(null)
const detailCheck = ref<PelicanCheck | null>(null)


let pollTimer: number | undefined
let refreshTimer: number | undefined

const accounts = computed(() => overview.value?.accounts ?? [])
const previewHtml = computed(() => detailCheck.value?.output_html || detailCheck.value?.output_excerpt || '')
const previewSrcdoc = computed(() => buildPreviewDocument(previewHtml.value))
const previewIsExcerpt = computed(() => !detailCheck.value?.output_html && !!detailCheck.value?.output_excerpt)
const columns = computed<Column[]>(() => [
  { key: 'account', label: t('admin.pelican.account') },
  { key: 'state', label: t('admin.pelican.state') },
  { key: 'last', label: t('admin.pelican.lastResult') },
  { key: 'streak', label: t('admin.pelican.streak') },
  { key: 'actions', label: t('admin.pelican.actions') },
])

async function reload() {
  loading.value = true
  error.value = ''
  try {
    overview.value = await pelicanAPI.getOverview()
  } catch (err) {
    error.value = extractApiErrorMessage(err, t('admin.pelican.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function runAccount(row: PelicanAccountOverview) {
  if (row.running || !row.supported) return
  error.value = ''
  try {
    await pelicanAPI.runAccount(row.account_id)
    await reload()
    schedulePoll()
  } catch (err) {
    error.value = extractApiErrorMessage(err, t('admin.pelican.runFailed'))
  }
}

async function openDetail(row: PelicanAccountOverview) {
  if (!row.last_check) return
  detailRow.value = row
  detailCheck.value = row.last_check
  detailError.value = ''
  showDetail.value = true
  detailLoading.value = true
  try {
    const checks = await pelicanAPI.listChecks(row.account_id, 10)
    const previewable = checks.find((check) => check.output_html || check.output_excerpt)
    if (previewable || checks.length > 0) {
      detailCheck.value = previewable || checks[0]
    }
  } catch (err) {
    detailError.value = extractApiErrorMessage(err, t('admin.pelican.detailLoadFailed'))
  } finally {
    detailLoading.value = false
  }
}

function closeDetail() {
  showDetail.value = false
}

function formatLatency(ms: number) {
  if (!ms || ms <= 0) return '-'
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(1)} s`
}

function schedulePoll() {
  if (pollTimer) window.clearTimeout(pollTimer)
  pollTimer = window.setTimeout(async () => {
    await reload()
    if (overview.value?.accounts.some((row) => row.running)) {
      schedulePoll()
    }
  }, 3000)
}

function verdictLabel(verdict: string, status?: string) {
  if (status === 'failed') return t('admin.pelican.failed')
  if (status === 'running') return t('admin.pelican.pending')
  if (verdict === 'healthy') return t('admin.pelican.healthy')
  if (verdict === 'degraded') return t('admin.pelican.degraded')
  if (verdict === 'inconclusive') return t('admin.pelican.inconclusive')
  if (verdict === 'judge_error') return t('admin.pelican.judgeError')
  return t('admin.pelican.pending')
}

function verdictClass(verdict: string, status?: string) {
  if (status === 'failed' || verdict === 'degraded' || verdict === 'judge_error') {
    return 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  }
  if (status === 'running' || !verdict) {
    return 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  }
  if (verdict === 'healthy') return 'bg-green-50 text-green-700 dark:bg-green-900/30 dark:text-green-300'
  return 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
}

function checkSummary(check: PelicanCheck) {
  const time = formatTime(check.created_at)
  if (check.status !== 'completed') return time
  return `${check.confidence || check.status} · ${time}`
}

function checkMessage(check: PelicanCheck) {
  if (check.status === 'failed') return check.error_message || check.reason
  return check.reason
}

function formatTime(value: string) {
  if (!value) return '-'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

onMounted(async () => {
  await reload()
  refreshTimer = window.setInterval(reload, 30000)
})

onUnmounted(() => {
  if (pollTimer) window.clearTimeout(pollTimer)
  if (refreshTimer) window.clearInterval(refreshTimer)
})
</script>