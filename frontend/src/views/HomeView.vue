<template>
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <div
    v-else
    class="relative min-h-screen overflow-hidden bg-[#fff8f4] text-gray-950 dark:bg-dark-950 dark:text-white"
  >
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div class="absolute left-1/2 top-0 h-[34rem] w-[34rem] -translate-x-1/2 rounded-full bg-primary-500/15 blur-3xl"></div>
      <div class="absolute -right-40 top-80 h-96 w-96 rounded-full bg-primary-400/15 blur-3xl"></div>
      <div class="absolute bottom-0 left-0 h-96 w-96 rounded-full bg-primary-700/10 blur-3xl"></div>
      <div
        class="absolute inset-0 bg-[linear-gradient(rgba(235,63,0,0.055)_1px,transparent_1px),linear-gradient(90deg,rgba(235,63,0,0.055)_1px,transparent_1px)] bg-[size:64px_64px] dark:bg-[linear-gradient(rgba(235,63,0,0.09)_1px,transparent_1px),linear-gradient(90deg,rgba(235,63,0,0.09)_1px,transparent_1px)]"
      ></div>
    </div>

    <header class="sticky top-0 z-30 border-b border-white/70 bg-white/75 backdrop-blur-xl dark:border-white/10 dark:bg-dark-950/70">
      <nav class="mx-auto flex max-w-7xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
        <router-link to="/" class="inline-flex items-center">
          <BrandLockup :name="siteName" :logo="siteLogo" :subtitle="t('home.brandSubtitle')" compact />
        </router-link>

        <div class="hidden items-center gap-8 text-sm font-medium text-gray-600 dark:text-dark-300 lg:flex">
          <a href="#capabilities" class="transition-colors hover:text-primary-600 dark:hover:text-primary-300">{{ t('home.nav.capabilities') }}</a>
          <a href="#models" class="transition-colors hover:text-primary-600 dark:hover:text-primary-300">{{ t('home.nav.models') }}</a>
          <a href="#billing" class="transition-colors hover:text-primary-600 dark:hover:text-primary-300">{{ t('home.nav.billing') }}</a>
        </div>

        <div class="flex items-center gap-2 sm:gap-3">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="hidden rounded-full p-2 text-gray-500 transition-colors hover:bg-white hover:text-primary-600 dark:text-dark-300 dark:hover:bg-white/10 dark:hover:text-primary-300 sm:inline-flex"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <button
            @click="toggleTheme"
            class="rounded-full p-2 text-gray-500 transition-colors hover:bg-white hover:text-primary-600 dark:text-dark-300 dark:hover:bg-white/10 dark:hover:text-primary-300"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            v-if="!isAuthenticated"
            to="/login"
            class="hidden rounded-full px-4 py-2 text-sm font-semibold text-gray-700 transition-colors hover:bg-white hover:text-primary-600 dark:text-dark-200 dark:hover:bg-white/10 dark:hover:text-primary-300 sm:inline-flex"
          >
            {{ t('home.login') }}
          </router-link>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/register'"
            class="inline-flex items-center rounded-full bg-primary-500 px-4 py-2 text-sm font-semibold text-white shadow-lg shadow-primary-500/25 transition hover:bg-primary-600 sm:px-5"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.register') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="relative z-10">
      <section class="mx-auto grid max-w-7xl gap-12 px-4 pb-16 pt-16 sm:px-6 lg:grid-cols-[1.05fr_0.95fr] lg:px-8 lg:pb-24 lg:pt-24">
        <div class="flex flex-col justify-center">
          <div class="mb-6 inline-flex w-fit items-center gap-2 rounded-full border border-primary-500/20 bg-white/80 px-4 py-2 text-sm font-medium text-primary-700 shadow-sm dark:bg-white/10 dark:text-primary-200">
            <span class="h-2 w-2 rounded-full bg-primary-500"></span>
            {{ t('home.heroEyebrow') }}
          </div>
          <h1 class="max-w-4xl text-5xl font-semibold tracking-tight text-gray-950 dark:text-white sm:text-6xl lg:text-7xl">
            {{ t('home.heroTitle') }}
          </h1>
          <p class="mt-6 max-w-2xl text-lg leading-8 text-gray-600 dark:text-dark-300 sm:text-xl">
            {{ siteSubtitle }}
          </p>
          <div class="mt-10 flex flex-col gap-3 sm:flex-row">
            <router-link
              :to="isAuthenticated ? dashboardPath : '/register'"
              class="btn btn-primary btn-lg shadow-xl shadow-primary-500/25"
            >
              {{ isAuthenticated ? t('home.goToDashboard') : t('home.primaryCta') }}
              <Icon name="chevronRight" size="sm" :stroke-width="2" />
            </router-link>
            <a href="#integration" class="btn btn-secondary btn-lg">
              {{ t('home.secondaryCta') }}
            </a>
          </div>
        </div>

        <div class="relative flex items-center justify-center lg:justify-end">
          <div class="w-full max-w-xl rounded-[2rem] border border-white/70 bg-white/85 p-4 shadow-2xl shadow-primary-950/10 backdrop-blur-xl dark:border-white/10 dark:bg-dark-900/80 dark:shadow-black/40">
            <div class="rounded-[1.5rem] bg-dark-950 p-5 text-white shadow-inner">
              <div class="mb-5 flex items-center justify-between border-b border-white/10 pb-4">
                <div class="flex items-center gap-2">
                  <span class="h-3 w-3 rounded-full bg-red-400"></span>
                  <span class="h-3 w-3 rounded-full bg-amber-400"></span>
                  <span class="h-3 w-3 rounded-full bg-emerald-400"></span>
                </div>
                <span class="font-mono text-xs text-white/40">api.longdao.ai</span>
              </div>
              <pre class="overflow-x-auto whitespace-pre-wrap font-mono text-sm leading-7 text-dark-100"><code>{{ codeSample }}</code></pre>
            </div>
            <div class="grid gap-3 pt-4 sm:grid-cols-3">
              <div v-for="metric in heroMetrics" :key="metric.label" class="rounded-2xl bg-primary-50 p-4 dark:bg-white/5">
                <p class="text-xl font-semibold text-primary-700 dark:text-primary-200">{{ metric.value }}</p>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ metric.label }}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section id="capabilities" class="mx-auto max-w-7xl px-4 py-14 sm:px-6 lg:px-8">
        <div class="mb-10 max-w-3xl">
          <p class="section-kicker">{{ t('home.capabilities.kicker') }}</p>
          <h2 class="section-title">{{ t('home.capabilities.title') }}</h2>
        </div>
        <div class="grid gap-5 md:grid-cols-2 xl:grid-cols-4">
          <article v-for="item in capabilityCards" :key="item.title" class="brand-card group">
            <div class="mb-5 flex h-12 w-12 items-center justify-center rounded-2xl bg-primary-500 text-white shadow-lg shadow-primary-500/25 transition group-hover:scale-105">
              <Icon :name="item.icon" size="lg" />
            </div>
            <h3 class="text-lg font-semibold text-gray-950 dark:text-white">{{ item.title }}</h3>
            <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ item.description }}</p>
          </article>
        </div>
      </section>

      <section id="models" class="mx-auto max-w-7xl px-4 py-14 sm:px-6 lg:px-8">
        <div class="rounded-[2rem] border border-primary-500/15 bg-white/80 p-6 shadow-xl shadow-primary-950/5 backdrop-blur dark:border-white/10 dark:bg-dark-900/70 lg:p-10">
          <div class="grid gap-10 lg:grid-cols-[0.8fr_1.2fr] lg:items-center">
            <div>
              <p class="section-kicker">{{ t('home.models.kicker') }}</p>
              <h2 class="section-title">{{ t('home.models.title') }}</h2>
              <p class="mt-4 text-sm leading-7 text-gray-600 dark:text-dark-300">{{ t('home.models.description') }}</p>
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <div v-for="model in modelCards" :key="model.title" class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-white/10 dark:bg-white/5">
                <p class="text-base font-semibold text-gray-950 dark:text-white">{{ model.title }}</p>
                <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ model.description }}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section id="integration" class="mx-auto max-w-7xl px-4 py-14 sm:px-6 lg:px-8">
        <div class="mb-10 max-w-3xl">
          <p class="section-kicker">{{ t('home.integration.kicker') }}</p>
          <h2 class="section-title">{{ t('home.integration.title') }}</h2>
        </div>
        <div class="grid gap-5 md:grid-cols-3">
          <article v-for="(step, index) in integrationSteps" :key="step.title" class="brand-card">
            <div class="mb-5 flex h-10 w-10 items-center justify-center rounded-full bg-primary-500 text-sm font-bold text-white">
              {{ index + 1 }}
            </div>
            <h3 class="text-lg font-semibold text-gray-950 dark:text-white">{{ step.title }}</h3>
            <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ step.description }}</p>
          </article>
        </div>
      </section>

      <section id="billing" class="mx-auto max-w-7xl px-4 py-14 sm:px-6 lg:px-8">
        <div class="grid gap-6 lg:grid-cols-[0.9fr_1.1fr] lg:items-stretch">
          <div class="brand-card flex flex-col justify-between bg-dark-950 text-white dark:bg-black">
            <div>
              <p class="section-kicker text-primary-200">{{ t('home.billing.kicker') }}</p>
              <h2 class="mt-3 text-3xl font-semibold tracking-tight text-white sm:text-4xl">{{ t('home.billing.title') }}</h2>
              <p class="mt-4 text-sm leading-7 text-white/65">{{ t('home.billing.description') }}</p>
            </div>
            <div class="mt-8 rounded-2xl border border-primary-300/20 bg-primary-500/10 p-5 text-sm leading-7 text-primary-50">
              {{ t('home.billing.rule') }}
            </div>
          </div>
          <div class="grid gap-5 sm:grid-cols-2">
            <article v-for="item in billingCards" :key="item.title" class="brand-card">
              <Icon :name="item.icon" size="lg" class="text-primary-600 dark:text-primary-300" />
              <h3 class="mt-4 text-lg font-semibold text-gray-950 dark:text-white">{{ item.title }}</h3>
              <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ item.description }}</p>
            </article>
          </div>
        </div>
      </section>

      <section class="mx-auto max-w-7xl px-4 py-14 sm:px-6 lg:px-8">
        <div class="rounded-[2rem] bg-primary-500 p-8 text-white shadow-2xl shadow-primary-500/25 sm:p-10 lg:flex lg:items-center lg:justify-between">
          <div>
            <h2 class="text-3xl font-semibold tracking-tight">{{ t('home.finalCta.title') }}</h2>
            <p class="mt-3 max-w-2xl text-sm leading-7 text-white/80">{{ t('home.finalCta.description') }}</p>
          </div>
          <router-link :to="isAuthenticated ? dashboardPath : '/register'" class="mt-6 inline-flex rounded-full bg-white px-6 py-3 text-sm font-semibold text-primary-700 shadow-lg transition hover:bg-primary-50 lg:mt-0">
            {{ isAuthenticated ? t('home.goToDashboard') : t('home.finalCta.button') }}
          </router-link>
        </div>
      </section>
    </main>

    <footer class="relative z-10 border-t border-primary-500/10 bg-white/60 py-8 backdrop-blur dark:border-white/10 dark:bg-dark-950/70">
      <div class="mx-auto flex max-w-7xl flex-col gap-4 px-4 text-sm text-gray-500 dark:text-dark-400 sm:flex-row sm:items-center sm:justify-between sm:px-6 lg:px-8">
        <BrandLockup :name="siteName" :logo="siteLogo" compact />
        <p>&copy; {{ currentYear }} {{ t('common.copyrightOwner') }}. {{ t('home.footer.allRightsReserved') }}</p>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import BrandLockup from '@/components/brand/BrandLockup.vue'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || '龙道 AI')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '/longdao-logo.png', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(
  () => appStore.cachedPublicSettings?.site_subtitle || t('home.heroDescription')
)
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isDark = ref(document.documentElement.classList.contains('dark'))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const currentYear = computed(() => new Date().getFullYear())

const codeSample = `curl https://api.longdao.ai/v1/chat/completions \\\n  -H "Authorization: Bearer $LONGDAO_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Hello"}]}'`

const heroMetrics = computed(() => [
  { value: t('home.metrics.compatibleValue'), label: t('home.metrics.compatibleLabel') },
  { value: t('home.metrics.billingValue'), label: t('home.metrics.billingLabel') },
  { value: t('home.metrics.securityValue'), label: t('home.metrics.securityLabel') }
])

const capabilityCards = computed(() => [
  { icon: 'link' as const, title: t('home.capabilities.unified.title'), description: t('home.capabilities.unified.description') },
  { icon: 'sync' as const, title: t('home.capabilities.routing.title'), description: t('home.capabilities.routing.description') },
  { icon: 'calculator' as const, title: t('home.capabilities.metering.title'), description: t('home.capabilities.metering.description') },
  { icon: 'shield' as const, title: t('home.capabilities.security.title'), description: t('home.capabilities.security.description') }
])

const modelCards = computed(() => [
  { title: t('home.models.openai.title'), description: t('home.models.openai.description') },
  { title: t('home.models.claude.title'), description: t('home.models.claude.description') },
  { title: t('home.models.gemini.title'), description: t('home.models.gemini.description') },
  { title: t('home.models.compatible.title'), description: t('home.models.compatible.description') }
])

const integrationSteps = computed(() => [
  { title: t('home.integration.createKey.title'), description: t('home.integration.createKey.description') },
  { title: t('home.integration.replaceBaseUrl.title'), description: t('home.integration.replaceBaseUrl.description') },
  { title: t('home.integration.sendRequest.title'), description: t('home.integration.sendRequest.description') }
])

const billingCards = computed(() => [
  { icon: 'dollar' as const, title: t('home.billing.rmb.title'), description: t('home.billing.rmb.description') },
  { icon: 'chart' as const, title: t('home.billing.transparent.title'), description: t('home.billing.transparent.description') },
  { icon: 'badge' as const, title: t('home.billing.packageFirst.title'), description: t('home.billing.packageFirst.description') },
  { icon: 'server' as const, title: t('home.billing.support.title'), description: t('home.billing.support.description') }
])

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.section-kicker {
  @apply text-sm font-semibold uppercase tracking-[0.28em] text-primary-600 dark:text-primary-300;
}

.section-title {
  @apply mt-3 text-3xl font-semibold tracking-tight text-gray-950 dark:text-white sm:text-4xl;
}

.brand-card {
  @apply rounded-[1.5rem] border border-white/70 bg-white/85 p-6 shadow-lg shadow-primary-950/5 backdrop-blur transition duration-300 dark:border-white/10 dark:bg-dark-900/75;
}

.brand-card:hover {
  @apply -translate-y-1 shadow-xl shadow-primary-950/10 dark:shadow-black/30;
}
</style>
