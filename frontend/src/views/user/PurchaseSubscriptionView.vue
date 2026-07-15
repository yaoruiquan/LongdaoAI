<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="relative overflow-hidden rounded-3xl bg-gray-950 px-6 py-8 text-white shadow-2xl shadow-gray-950/10 md:px-8 lg:px-10">
        <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_85%_20%,rgba(235,63,0,0.28),transparent_34%),linear-gradient(rgba(255,255,255,0.035)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.035)_1px,transparent_1px)] bg-[size:auto,36px_36px,36px_36px]"></div>
        <div class="relative grid gap-7 lg:grid-cols-[1fr_auto] lg:items-end">
          <div>
            <div class="mb-3 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs font-medium text-primary-200">
              <span class="h-1.5 w-1.5 rounded-full bg-primary-500"></span>
              {{ t('longdao.plans.badge') }}
            </div>
            <h1 class="text-3xl font-semibold tracking-tight md:text-4xl">{{ t('longdao.plans.title') }}</h1>
            <p class="mt-3 max-w-2xl text-sm leading-6 text-gray-300">{{ t('longdao.plans.description') }}</p>
          </div>
          <div class="rounded-2xl border border-white/10 bg-white/5 px-5 py-4 backdrop-blur">
            <p class="text-xs uppercase tracking-[0.16em] text-gray-400">{{ t('longdao.plans.balance') }}</p>
            <p class="mt-1 text-2xl font-semibold">¥{{ formatMoney(user?.balance || 0) }}</p>
            <router-link to="/recharge" class="mt-2 inline-flex text-xs font-medium text-primary-300 hover:text-primary-200">
              {{ t('longdao.plans.goRecharge') }} →
            </router-link>
          </div>
        </div>
      </section>

      <RuleNotice :title="t('longdao.plans.paymentPendingTitle')" :description="t('longdao.plans.paymentPendingDescription')">
        <template #icon>!</template>
      </RuleNotice>

      <section>
        <div class="mb-4 flex items-end justify-between gap-4">
          <div>
            <h2 class="text-xl font-semibold text-gray-950 dark:text-white">{{ t('longdao.plans.availablePlans') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('longdao.plans.availablePlansDesc') }}</p>
          </div>
          <span class="badge badge-warning">{{ t('longdao.plans.frontendPreview') }}</span>
        </div>
        <div class="grid gap-5 lg:grid-cols-3">
          <article
            v-for="(plan, index) in planCards"
            :key="plan.title"
            class="relative flex min-h-[25rem] flex-col overflow-hidden rounded-3xl border bg-white p-6 shadow-card transition-transform duration-200 hover:-translate-y-1 dark:bg-dark-900"
            :class="index === 1 ? 'border-primary-300 ring-1 ring-primary-200 dark:border-primary-800 dark:ring-primary-900' : 'border-gray-200 dark:border-dark-700'"
          >
            <div v-if="index === 1" class="absolute right-5 top-5 rounded-full bg-primary-50 px-2.5 py-1 text-[11px] font-semibold text-primary-700 dark:bg-primary-950/60 dark:text-primary-300">
              {{ t('longdao.plans.recommended') }}
            </div>
            <div class="flex h-11 w-11 items-center justify-center rounded-2xl bg-primary-50 text-lg font-semibold text-primary-700 dark:bg-primary-950/40 dark:text-primary-300">{{ index + 1 }}</div>
            <h3 class="mt-5 text-xl font-semibold text-gray-950 dark:text-white">{{ plan.title }}</h3>
            <p class="mt-2 min-h-12 text-sm leading-6 text-gray-500 dark:text-dark-400">{{ plan.description }}</p>
            <div class="my-5 border-y border-gray-100 py-4 dark:border-dark-700">
              <p class="text-xs font-medium uppercase tracking-[0.14em] text-gray-400">{{ t('longdao.plans.price') }}</p>
              <p class="mt-1 text-2xl font-semibold text-gray-950 dark:text-white">{{ t('longdao.plans.configuredByOperations') }}</p>
            </div>
            <ul class="flex-1 space-y-3">
              <li v-for="feature in plan.features" :key="feature" class="flex gap-2 text-sm text-gray-600 dark:text-dark-300">
                <span class="mt-1 flex h-4 w-4 flex-shrink-0 items-center justify-center rounded-full bg-primary-50 text-[10px] text-primary-700 dark:bg-primary-950/50 dark:text-primary-300">✓</span>
                <span>{{ feature }}</span>
              </li>
            </ul>
            <button type="button" class="btn btn-primary mt-6 w-full" disabled>
              {{ t('longdao.plans.paymentConnecting') }}
            </button>
          </article>
        </div>
      </section>

      <section class="rounded-3xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900 md:p-8">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 class="text-xl font-semibold text-gray-950 dark:text-white">{{ t('longdao.plans.currentSubscriptions') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('longdao.plans.currentSubscriptionsDesc') }}</p>
          </div>
          <router-link to="/subscriptions" class="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400">{{ t('longdao.plans.viewDetails') }} →</router-link>
        </div>

        <div v-if="loading" class="flex justify-center py-12"><LoadingSpinner /></div>
        <div v-else-if="subscriptions.length" class="mt-5 grid gap-4 md:grid-cols-2">
          <div v-for="subscription in subscriptions" :key="subscription.id" class="rounded-2xl border border-gray-200 bg-gray-50 p-5 dark:border-dark-700 dark:bg-dark-800/60">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="font-semibold text-gray-950 dark:text-white">{{ subscription.group?.name || t('longdao.plans.unnamedPlan') }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ subscription.group?.description || t('longdao.plans.subscriptionActive') }}</p>
              </div>
              <span class="badge" :class="subscription.status === 'active' ? 'badge-success' : 'badge-gray'">{{ subscription.status }}</span>
            </div>
            <div class="mt-4 grid grid-cols-2 gap-3 text-xs">
              <div class="rounded-xl bg-white p-3 dark:bg-dark-900">
                <p class="text-gray-400">{{ t('longdao.plans.monthlyQuota') }}</p>
                <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ formatQuota(subscription.group?.monthly_limit_usd) }}</p>
              </div>
              <div class="rounded-xl bg-white p-3 dark:bg-dark-900">
                <p class="text-gray-400">{{ t('longdao.plans.multiplier') }}</p>
                <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ subscription.group?.rate_multiplier ?? '--' }}×</p>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="mt-5 rounded-2xl border border-dashed border-gray-300 px-6 py-10 text-center dark:border-dark-600">
          <p class="font-medium text-gray-900 dark:text-white">{{ t('longdao.plans.noSubscription') }}</p>
          <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('longdao.plans.noSubscriptionDesc') }}</p>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import RuleNotice from '@/components/common/RuleNotice.vue'
import subscriptionsAPI from '@/api/subscriptions'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import type { UserSubscription } from '@/types'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const user = computed(() => authStore.user)
const subscriptions = ref<UserSubscription[]>([])
const loading = ref(true)

const planCards = computed(() => [
  {
    title: t('longdao.plans.starter.title'),
    description: t('longdao.plans.starter.description'),
    features: [t('longdao.plans.starter.feature1'), t('longdao.plans.starter.feature2'), t('longdao.plans.starter.feature3')]
  },
  {
    title: t('longdao.plans.professional.title'),
    description: t('longdao.plans.professional.description'),
    features: [t('longdao.plans.professional.feature1'), t('longdao.plans.professional.feature2'), t('longdao.plans.professional.feature3')]
  },
  {
    title: t('longdao.plans.enterprise.title'),
    description: t('longdao.plans.enterprise.description'),
    features: [t('longdao.plans.enterprise.feature1'), t('longdao.plans.enterprise.feature2'), t('longdao.plans.enterprise.feature3')]
  }
])

const formatMoney = (value: number) => Number(value || 0).toFixed(2)
const formatQuota = (value: number | null | undefined) => value ? `$${value.toFixed(2)}` : t('longdao.plans.unlimitedOrConfigured')

onMounted(async () => {
  try {
    await authStore.refreshUser()
    subscriptions.value = await subscriptionsAPI.getMySubscriptions()
  } catch (error) {
    console.error('Failed to load subscription center:', error)
    appStore.showError(t('userSubscriptions.failedToLoad'))
  } finally {
    loading.value = false
  }
})
</script>
