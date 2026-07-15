<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <header class="space-y-2">
        <p class="text-sm font-semibold uppercase tracking-wide text-primary-600 dark:text-primary-300">
          {{ t('longdao.finance.orders.badge') }}
        </p>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">
          {{ t('longdao.finance.orders.title') }}
        </h1>
        <p class="max-w-3xl text-sm leading-6 text-gray-600 dark:text-dark-300 sm:text-base">
          {{ t('longdao.finance.orders.description') }}
        </p>
      </header>

      <section class="card p-6 sm:p-8">
        <EmptyState
          :title="t('longdao.finance.orders.emptyTitle')"
          :description="t('longdao.finance.orders.emptyDescription')"
          :action-icon="false"
        >
          <template #icon>
            <div class="flex h-10 w-10 items-center justify-center text-gray-400 dark:text-dark-400">
              <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M8.25 6.75h7.5m-7.5 4.5h7.5m-7.5 4.5h4.5M5.25 3.75h13.5A1.5 1.5 0 0 1 20.25 5.25v13.5a1.5 1.5 0 0 1-1.5 1.5H5.25a1.5 1.5 0 0 1-1.5-1.5V5.25a1.5 1.5 0 0 1 1.5-1.5Z" />
              </svg>
            </div>
          </template>
        </EmptyState>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()
</script>
