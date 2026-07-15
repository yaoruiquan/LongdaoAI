<template>
  <div
    class="relative min-h-screen overflow-hidden bg-[#fff8f4] px-4 py-6 text-gray-950 dark:bg-dark-950 dark:text-white sm:px-6 lg:px-8"
  >
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div class="absolute -left-24 top-20 h-72 w-72 rounded-full bg-primary-500/20 blur-3xl"></div>
      <div class="absolute -right-24 bottom-10 h-96 w-96 rounded-full bg-primary-400/20 blur-3xl"></div>
      <div
        class="absolute inset-0 bg-[linear-gradient(rgba(235,63,0,0.06)_1px,transparent_1px),linear-gradient(90deg,rgba(235,63,0,0.06)_1px,transparent_1px)] bg-[size:56px_56px] dark:bg-[linear-gradient(rgba(235,63,0,0.10)_1px,transparent_1px),linear-gradient(90deg,rgba(235,63,0,0.10)_1px,transparent_1px)]"
      ></div>
    </div>

    <div class="relative z-10 mx-auto flex min-h-[calc(100vh-3rem)] w-full max-w-6xl items-center">
      <div
        class="grid w-full overflow-hidden rounded-[2rem] border border-white/70 bg-white/80 shadow-2xl shadow-primary-950/10 backdrop-blur-xl dark:border-white/10 dark:bg-dark-900/80 dark:shadow-black/40 lg:grid-cols-[1.05fr_0.95fr]"
      >
        <aside
          class="relative hidden min-h-[720px] flex-col justify-between overflow-hidden bg-gradient-to-br from-dark-950 via-dark-900 to-[#260901] p-10 text-white lg:flex"
        >
          <div class="absolute inset-0 bg-mesh-gradient opacity-80"></div>
          <div class="absolute -right-24 top-20 h-64 w-64 rounded-full bg-primary-500/30 blur-3xl"></div>
          <div class="absolute bottom-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-primary-400 to-transparent"></div>

          <div class="relative">
            <BrandLockup :name="siteName" :logo="siteLogo" :subtitle="t('auth.brandEyebrow')" />
            <div class="mt-16 max-w-xl">
              <p class="mb-4 text-sm font-semibold uppercase tracking-[0.32em] text-primary-200">
                {{ t('auth.platformTagline') }}
              </p>
              <h1 class="text-4xl font-semibold tracking-tight text-white xl:text-5xl">
                {{ t('auth.panelTitle') }}
              </h1>
              <p class="mt-6 text-base leading-8 text-white/70">
                {{ t('auth.panelDescription') }}
              </p>
            </div>
          </div>

          <div class="relative space-y-6">
            <div class="grid grid-cols-3 gap-3">
              <div
                v-for="metric in metrics"
                :key="metric.value"
                class="rounded-2xl border border-white/10 bg-white/10 p-4 backdrop-blur"
              >
                <p class="text-2xl font-semibold text-white">{{ metric.value }}</p>
                <p class="mt-1 text-xs text-white/55">{{ metric.label }}</p>
              </div>
            </div>
            <div class="rounded-3xl border border-primary-300/20 bg-primary-500/10 p-5">
              <p class="text-sm font-medium text-primary-100">{{ t('auth.billingPrincipleTitle') }}</p>
              <p class="mt-2 text-sm leading-6 text-white/65">{{ t('auth.billingPrincipleDesc') }}</p>
            </div>
          </div>
        </aside>

        <main class="flex min-h-[640px] flex-col p-6 sm:p-8 lg:p-10">
          <div class="mb-8 flex items-center justify-between lg:hidden">
            <BrandLockup :name="siteName" :logo="siteLogo" compact />
          </div>

          <div class="flex flex-1 items-center justify-center">
            <div class="w-full max-w-md">
              <slot />
              <div class="mt-6 text-center text-sm">
                <slot name="footer" />
              </div>
            </div>
          </div>

          <div class="mt-8 text-center text-xs text-gray-400 dark:text-dark-500">
            &copy; {{ currentYear }} {{ t('common.copyrightOwner') }}. All rights reserved.
          </div>
        </main>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import BrandLockup from '@/components/brand/BrandLockup.vue'

const { t } = useI18n()
const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || '龙道 AI')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '/longdao-logo.png', { allowRelative: true, allowDataUrl: true }))
const currentYear = computed(() => new Date().getFullYear())

const metrics = computed(() => [
  { value: 'OpenAI', label: t('auth.metrics.compatible') },
  { value: 'Claude', label: t('auth.metrics.routing') },
  { value: '¥ / $', label: t('auth.metrics.billing') }
])

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>
