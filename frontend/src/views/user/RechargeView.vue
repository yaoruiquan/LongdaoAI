<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <header class="space-y-2">
        <p class="text-sm font-semibold uppercase tracking-wide text-primary-600 dark:text-primary-300">
          {{ t('longdao.finance.recharge.badge') }}
        </p>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">
          {{ t('longdao.finance.recharge.title') }}
        </h1>
        <p class="max-w-3xl text-sm leading-6 text-gray-600 dark:text-dark-300 sm:text-base">
          {{ t('longdao.finance.recharge.description') }}
        </p>
      </header>

      <div class="grid gap-6 lg:grid-cols-[20rem_1fr]">
        <section class="card overflow-hidden">
          <div class="bg-gradient-to-br from-primary-500 to-primary-700 p-6 text-white">
            <p class="text-sm font-medium text-primary-100">
              {{ t('longdao.finance.recharge.currentBalance') }}
            </p>
            <p class="mt-3 text-4xl font-bold">
              ¥{{ currentBalance }}
            </p>
            <p class="mt-3 text-sm text-primary-100">
              {{ t('longdao.finance.recharge.balanceHint') }}
            </p>
          </div>
          <div class="space-y-3 p-5 text-sm text-gray-600 dark:text-dark-300">
            <div class="flex items-center justify-between gap-4">
              <span>{{ t('longdao.finance.recharge.accountStatus') }}</span>
              <span class="rounded-full bg-gray-100 px-3 py-1 font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200">
                {{ t('longdao.finance.recharge.notConnected') }}
              </span>
            </div>
            <div class="flex items-center justify-between gap-4">
              <span>{{ t('longdao.finance.recharge.currency') }}</span>
              <span class="font-medium text-gray-900 dark:text-white">CNY</span>
            </div>
          </div>
        </section>

        <section class="card p-5 sm:p-6">
          <div class="space-y-6">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('longdao.finance.recharge.amountTitle') }}
              </h2>
              <div class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
                <button
                  v-for="amount in presetAmounts"
                  :key="amount"
                  type="button"
                  :class="[
                    'rounded-2xl border px-4 py-4 text-center text-lg font-semibold transition-colors',
                    selectedAmount === amount && !customAmount
                      ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200'
                      : 'border-gray-200 bg-white text-gray-800 hover:border-primary-300 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-100'
                  ]"
                  @click="selectPreset(amount)"
                >
                  ¥{{ amount }}
                </button>
              </div>
            </div>

            <label class="block">
              <span class="input-label">{{ t('longdao.finance.recharge.customAmount') }}</span>
              <input
                v-model="customAmount"
                class="input mt-2"
                type="number"
                min="1"
                step="1"
                inputmode="decimal"
                :placeholder="t('longdao.finance.recharge.customAmountPlaceholder')"
              />
            </label>

            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('longdao.finance.recharge.paymentMethodTitle') }}
              </h2>
              <div class="mt-4 grid gap-3 sm:grid-cols-2">
                <div v-for="methodKey in paymentMethodKeys" :key="methodKey" class="rounded-2xl border border-dashed border-gray-200 p-4 dark:border-dark-700">
                  <p class="font-medium text-gray-900 dark:text-white">{{ t(methodKey) }}</p>
                  <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                    {{ t('longdao.finance.recharge.paymentMethodPending') }}
                  </p>
                </div>
              </div>
            </div>

            <div class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200">
              {{ t('longdao.finance.recharge.integrationNotice') }}
            </div>

            <button type="button" class="btn btn-primary w-full py-3" disabled aria-disabled="true">
              {{ t('longdao.finance.recharge.disabledButton') }}
            </button>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const authStore = useAuthStore()

const presetAmounts = [100, 300, 500, 1000]
const paymentMethodKeys = [
  'longdao.finance.recharge.paymentMethods.card',
  'longdao.finance.recharge.paymentMethods.wallet'
]

const selectedAmount = ref(presetAmounts[1])
const customAmount = ref('')

const currentBalance = computed(() => (authStore.user?.balance ?? 0).toFixed(2))

function selectPreset(amount: number) {
  selectedAmount.value = amount
  customAmount.value = ''
}
</script>
