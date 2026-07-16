<template>
  <section class="overflow-hidden rounded-3xl border border-dashed border-primary-200 bg-white shadow-soft dark:border-primary-900/50 dark:bg-dark-800/60">
    <div class="grid gap-0 lg:grid-cols-[1fr_20rem]">
      <div class="space-y-6 p-6 sm:p-8">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-start">
          <div class="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
            <svg class="h-7 w-7" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.8" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" d="M10.5 6h3m-7 4.5h11m-9 4.5h7m-9.75 4.5h12.5A1.75 1.75 0 0 0 20 17.75V6.25A1.75 1.75 0 0 0 18.25 4.5H5.75A1.75 1.75 0 0 0 4 6.25v11.5c0 .966.784 1.75 1.75 1.75Z" />
            </svg>
          </div>
          <div class="min-w-0 space-y-2">
            <p class="text-sm font-semibold uppercase tracking-wide text-primary-600 dark:text-primary-300">
              {{ t(badgeKey) }}
            </p>
            <h1 class="text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">
              {{ t(titleKey) }}
            </h1>
            <p class="max-w-2xl text-sm leading-6 text-gray-600 dark:text-dark-300 sm:text-base">
              {{ t(descriptionKey) }}
            </p>
          </div>
        </div>

        <div v-if="capabilityKeys.length" class="rounded-2xl bg-gray-50 p-5 dark:bg-dark-900/40">
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t(capabilitiesTitleKey) }}
          </h2>
          <ul class="mt-4 grid gap-3 sm:grid-cols-2">
            <li v-for="capabilityKey in capabilityKeys" :key="capabilityKey" class="flex gap-3 text-sm text-gray-600 dark:text-dark-300">
              <span class="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary-100 text-primary-600 dark:bg-primary-900/40 dark:text-primary-300">
                <svg class="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2" aria-hidden="true">
                  <path stroke-linecap="round" stroke-linejoin="round" d="m5 13 4 4L19 7" />
                </svg>
              </span>
              <span>{{ t(capabilityKey) }}</span>
            </li>
          </ul>
        </div>
      </div>

      <aside class="border-t border-primary-100 bg-primary-50/70 p-6 dark:border-primary-900/40 dark:bg-primary-950/20 lg:border-l lg:border-t-0">
        <div class="rounded-2xl bg-white/80 p-5 shadow-sm dark:bg-dark-800/80">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t(statusTitleKey) }}
          </p>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
            {{ t(statusDescriptionKey) }}
          </p>
          <div class="mt-5 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200">
            {{ t(disabledNoticeKey) }}
          </div>
        </div>
      </aside>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

interface Props {
  titleKey: string
  descriptionKey: string
  capabilityKeys?: string[]
  badgeKey?: string
  capabilitiesTitleKey?: string
  statusTitleKey?: string
  statusDescriptionKey?: string
  disabledNoticeKey?: string
}

withDefaults(defineProps<Props>(), {
  capabilityKeys: () => [],
  badgeKey: 'longdao.adminConstruction.badge',
  capabilitiesTitleKey: 'longdao.adminConstruction.futureCapabilities',
  statusTitleKey: 'longdao.adminConstruction.statusTitle',
  statusDescriptionKey: 'longdao.adminConstruction.statusDescription',
  disabledNoticeKey: 'longdao.adminConstruction.disabledNotice'
})

const { t } = useI18n()
</script>
