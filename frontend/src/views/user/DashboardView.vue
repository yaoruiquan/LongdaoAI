<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="relative overflow-hidden rounded-3xl border border-gray-200 bg-white p-6 shadow-card dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <div class="pointer-events-none absolute right-0 top-0 h-48 w-48 translate-x-1/3 -translate-y-1/3 rounded-full bg-primary-500/10 blur-3xl"></div>
        <div class="relative flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold text-primary-700 dark:bg-primary-950/40 dark:text-primary-300">
              <span class="h-1.5 w-1.5 rounded-full bg-primary-500"></span>
              {{ t('longdao.dashboard.badge') }}
            </div>
            <h1 class="text-3xl font-semibold tracking-tight text-gray-950 dark:text-white">
              {{ t('longdao.dashboard.welcome', { name: user?.username || user?.email || t('longdao.dashboard.developer') }) }}
            </h1>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ t('longdao.dashboard.description') }}
            </p>
          </div>
          <div class="flex flex-wrap gap-3">
            <router-link to="/keys" class="btn btn-primary">{{ t('longdao.dashboard.createKey') }}</router-link>
            <router-link to="/purchase" class="btn btn-secondary">{{ t('longdao.dashboard.viewPlans') }}</router-link>
          </div>
        </div>
      </section>

      <RuleNotice
        :title="t('longdao.dashboard.billingRuleTitle')"
        :description="t('longdao.dashboard.billingRuleDescription')"
      >
        <template #icon>¥</template>
      </RuleNotice>

      <div v-if="loading" class="flex items-center justify-center py-12"><LoadingSpinner /></div>
      <div v-else-if="error" class="card p-8 text-center">
        <p class="font-medium text-gray-900 dark:text-white">{{ t('longdao.dashboard.loadFailed') }}</p>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ error }}</p>
        <button type="button" class="btn btn-secondary mt-5" @click="refreshAll">{{ t('common.retry') }}</button>
      </div>
      <template v-else-if="stats">
        <UserDashboardStats :stats="stats" :balance="user?.balance || 0" :is-simple="authStore.isSimpleMode" />
        <UserDashboardCharts v-model:startDate="startDate" v-model:endDate="endDate" v-model:granularity="granularity" :loading="loadingCharts" :trend="trendData" :models="modelStats" @dateRangeChange="loadCharts" @granularityChange="loadCharts" @refresh="refreshAll" />
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div class="lg:col-span-2"><UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" /></div>
          <div class="lg:col-span-1"><UserDashboardQuickActions /></div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import RuleNotice from '@/components/common/RuleNotice.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { UsageLog, TrendDataPoint, ModelStat } from '@/types'

const { t } = useI18n()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const loadingUsage = ref(false)
const loadingCharts = ref(false)
const error = ref('')
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])

const formatLD = (d: Date) => d.toISOString().split('T')[0]
const startDate = ref(formatLD(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatLD(new Date()))
const granularity = ref('day')

const loadStats = async () => {
  loading.value = true
  error.value = ''
  try {
    await authStore.refreshUser()
    stats.value = await usageAPI.getDashboardStats()
  } catch (caught) {
    console.error('Failed to load dashboard stats:', caught)
    error.value = caught instanceof Error ? caught.message : t('longdao.dashboard.loadFailed')
  } finally {
    loading.value = false
  }
}
const loadCharts = async () => {
  loadingCharts.value = true
  try {
    const res = await Promise.all([
      usageAPI.getDashboardTrend({ start_date: startDate.value, end_date: endDate.value, granularity: granularity.value as 'day' | 'hour' }),
      usageAPI.getDashboardModels({ start_date: startDate.value, end_date: endDate.value })
    ])
    trendData.value = res[0].trend || []
    modelStats.value = res[1].models || []
  } catch (caught) {
    console.error('Failed to load charts:', caught)
  } finally {
    loadingCharts.value = false
  }
}
const loadRecent = async () => {
  loadingUsage.value = true
  try {
    const res = await usageAPI.getByDateRange(startDate.value, endDate.value)
    recentUsage.value = res.items.slice(0, 5)
  } catch (caught) {
    console.error('Failed to load recent usage:', caught)
  } finally {
    loadingUsage.value = false
  }
}
const refreshAll = () => {
  void loadStats()
  void loadCharts()
  void loadRecent()
}

onMounted(refreshAll)
</script>
