<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <header class="space-y-2">
        <p class="text-sm font-semibold uppercase tracking-wide text-primary-600 dark:text-primary-300">
          {{ t('longdao.plans.badge') }}
        </p>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">
          {{ t('longdao.plans.title') }}
        </h1>
        <p class="max-w-3xl text-sm leading-6 text-gray-600 dark:text-dark-300 sm:text-base">
          {{ t('longdao.plans.description') }}
        </p>
      </header>

      <section class="card flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between sm:p-6">
        <div>
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('longdao.plans.balance') }}</p>
          <p class="mt-1 text-3xl font-bold text-gray-900 dark:text-white">¥{{ balance }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('longdao.plans.billingPriority') }}</p>
        </div>
        <RouterLink to="/recharge" class="btn btn-secondary self-start sm:self-auto">
          {{ t('longdao.plans.goRecharge') }}
        </RouterLink>
      </section>

      <RuleNotice
        :title="t('longdao.plans.paymentPendingTitle')"
        :description="t('longdao.plans.paymentPendingDescription')"
      />

      <section class="space-y-4">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('longdao.plans.availablePlans') }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('longdao.plans.availablePlansDesc') }}
          </p>
        </div>

        <div class="grid gap-4 md:grid-cols-3">
          <article
            v-for="plan in plans"
            :key="plan.key"
            class="card relative flex flex-col gap-4 p-5 sm:p-6"
            :class="plan.recommended ? 'ring-2 ring-primary-500' : ''"
          >
            <span
              v-if="plan.recommended"
              class="absolute right-4 top-4 rounded-full bg-primary-100 px-3 py-1 text-xs font-semibold text-primary-700 dark:bg-primary-900/40 dark:text-primary-200"
            >
              {{ t('longdao.plans.recommended') }}
            </span>
            <div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t(`longdao.plans.${plan.key}.title`) }}
              </h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                {{ t(`longdao.plans.${plan.key}.description`) }}
              </p>
            </div>
            <ul class="space-y-2 text-sm text-gray-600 dark:text-dark-300">
              <li v-for="feature in plan.features" :key="feature" class="flex items-start gap-2">
                <span class="mt-1 h-1.5 w-1.5 flex-shrink-0 rounded-full bg-primary-500"></span>
                <span>{{ t(feature) }}</span>
              </li>
            </ul>
            <div class="mt-auto space-y-3 pt-2">
              <div class="flex items-center justify-between text-sm">
                <span class="text-gray-500 dark:text-dark-400">{{ t('longdao.plans.price') }}</span>
                <span class="font-medium text-gray-900 dark:text-white">
                  {{ t('longdao.plans.configuredByOperations') }}
                </span>
              </div>
              <button type="button" class="btn btn-primary w-full" disabled aria-disabled="true">
                {{ t('longdao.plans.paymentConnecting') }}
              </button>
            </div>
          </article>
        </div>
      </section>

      <section class="card p-5 sm:p-6">
        <div class="flex items-center justify-between gap-4">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('longdao.plans.currentSubscriptions') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ t('longdao.plans.currentSubscriptionsDesc') }}
            </p>
          </div>
        </div>

        <div class="mt-4">
          <LoadingSpinner v-if="loading" />
          <div
            v-else-if="subscriptions.length === 0"
            class="rounded-2xl border border-dashed border-gray-200 p-6 text-center dark:border-dark-700"
          >
            <p class="font-medium text-gray-900 dark:text-white">{{ t('longdao.plans.noSubscription') }}</p>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('longdao.plans.noSubscriptionDesc') }}</p>
          </div>
          <ul v-else class="space-y-3">
            <li
              v-for="subscription in subscriptions"
              :key="subscription.id"
              class="rounded-2xl border border-gray-200 p-4 dark:border-dark-700"
            >
              <div class="flex items-center justify-between gap-4">
                <span class="font-medium text-gray-900 dark:text-white">
                  {{ subscription.group?.name || t('longdao.plans.unnamedPlan') }}
                </span>
                <span class="rounded-full bg-emerald-100 px-3 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-200">
                  {{ t('longdao.plans.subscriptionActive') }}
                </span>
              </div>
            </li>
          </ul>
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

const loading = ref(true)
const subscriptions = ref<UserSubscription[]>([])

const balance = computed(() => (authStore.user?.balance ?? 0).toFixed(2))

const plans = [
  {
    key: 'starter',
    recommended: false,
    features: [
      'longdao.plans.starter.feature1',
      'longdao.plans.starter.feature2',
      'longdao.plans.starter.feature3'
    ]
  },
  {
    key: 'professional',
    recommended: true,
    features: [
      'longdao.plans.professional.feature1',
      'longdao.plans.professional.feature2',
      'longdao.plans.professional.feature3'
    ]
  },
  {
    key: 'enterprise',
    recommended: false,
    features: [
      'longdao.plans.enterprise.feature1',
      'longdao.plans.enterprise.feature2',
      'longdao.plans.enterprise.feature3'
    ]
  }
] as const

onMounted(async () => {
  try {
    await authStore.refreshUser()
    subscriptions.value = await subscriptionsAPI.getMySubscriptions()
  } catch {
    appStore.showError(t('longdao.dashboard.loadFailed'))
  } finally {
    loading.value = false
  }
})
</script>
