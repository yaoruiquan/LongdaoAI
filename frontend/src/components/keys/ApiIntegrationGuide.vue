<template>
  <section class="overflow-hidden rounded-3xl border border-gray-200 bg-white shadow-card dark:border-dark-700 dark:bg-dark-900">
    <div class="grid gap-6 border-b border-gray-100 bg-gradient-to-br from-primary-50/80 via-white to-white px-6 py-7 dark:border-dark-700 dark:from-primary-950/25 dark:via-dark-900 dark:to-dark-900 lg:grid-cols-[1.25fr_0.75fr] lg:px-8">
      <div>
        <div class="mb-3 inline-flex items-center gap-2 rounded-full border border-primary-200 bg-white/80 px-3 py-1 text-xs font-semibold text-primary-700 dark:border-primary-800 dark:bg-dark-900/70 dark:text-primary-300">
          <span class="h-1.5 w-1.5 rounded-full bg-primary-500"></span>
          {{ t('longdao.apiGuide.badge') }}
        </div>
        <h2 class="text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">
          {{ t('longdao.apiGuide.title') }}
        </h2>
        <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-dark-300">
          {{ t('longdao.apiGuide.description') }}
        </p>
      </div>

      <div class="rounded-2xl border border-gray-200 bg-white/90 p-4 dark:border-dark-700 dark:bg-dark-950/70">
        <p class="text-xs font-semibold uppercase tracking-[0.18em] text-gray-400">Base URL</p>
        <div class="mt-2 flex items-center gap-2">
          <code class="min-w-0 flex-1 truncate rounded-xl bg-gray-950 px-3 py-2.5 text-xs text-gray-100">{{ normalizedBaseUrl }}</code>
          <button type="button" class="btn btn-secondary btn-sm flex-shrink-0" @click="copy(normalizedBaseUrl)">
            {{ t('longdao.apiGuide.copy') }}
          </button>
        </div>
        <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-dark-400">
          {{ t('longdao.apiGuide.authHint') }}
        </p>
      </div>
    </div>

    <div class="p-6 lg:p-8">
      <div class="flex flex-wrap gap-2" role="tablist" :aria-label="t('longdao.apiGuide.tabsLabel')">
        <button
          v-for="example in examples"
          :key="example.id"
          type="button"
          class="rounded-xl px-4 py-2 text-sm font-medium transition-colors"
          :class="activeTab === example.id
            ? 'bg-primary-600 text-white shadow-sm shadow-primary-500/20'
            : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-800 dark:text-dark-300 dark:hover:bg-dark-700'"
          @click="activeTab = example.id"
        >
          {{ example.label }}
        </button>
      </div>

      <div class="relative mt-4 overflow-hidden rounded-2xl border border-gray-800 bg-[#101113] shadow-2xl shadow-gray-950/10">
        <div class="flex items-center justify-between border-b border-white/10 px-4 py-3">
          <div class="flex items-center gap-1.5">
            <span class="h-2.5 w-2.5 rounded-full bg-[#ff5f57]"></span>
            <span class="h-2.5 w-2.5 rounded-full bg-[#febc2e]"></span>
            <span class="h-2.5 w-2.5 rounded-full bg-[#28c840]"></span>
          </div>
          <button type="button" class="text-xs font-medium text-gray-400 transition-colors hover:text-white" @click="copy(activeExample.code)">
            {{ t('longdao.apiGuide.copyCode') }}
          </button>
        </div>
        <pre class="max-h-[28rem] overflow-auto p-5 text-[13px] leading-6 text-gray-200"><code>{{ activeExample.code }}</code></pre>
      </div>

      <div class="mt-5 grid gap-3 md:grid-cols-3">
        <div v-for="item in safetyItems" :key="item.title" class="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/60">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ item.title }}</p>
          <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ item.description }}</p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'

const props = defineProps<{ baseUrl?: string }>()
const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const activeTab = ref('openai')

const normalizedBaseUrl = computed(() => {
  const configured = props.baseUrl?.trim().replace(/\/$/, '')
  if (configured) return configured
  if (typeof window !== 'undefined') return `${window.location.origin}/v1`
  return 'https://your-domain.example/v1'
})

const examples = computed(() => [
  {
    id: 'openai',
    label: 'OpenAI SDK',
    code: `from openai import OpenAI\n\nclient = OpenAI(\n    api_key="YOUR_LONGDAO_API_KEY",\n    base_url="${normalizedBaseUrl.value}"\n)\n\nresponse = client.chat.completions.create(\n    model="gpt-4o-mini",\n    messages=[{"role": "user", "content": "Hello, Longdao AI"}]\n)\n\nprint(response.choices[0].message.content)`
  },
  {
    id: 'claude',
    label: 'Claude SDK',
    code: `import anthropic\n\nclient = anthropic.Anthropic(\n    api_key="YOUR_LONGDAO_API_KEY",\n    base_url="${normalizedBaseUrl.value.replace(/\/v1$/, '')}"\n)\n\nmessage = client.messages.create(\n    model="claude-sonnet-4-5",\n    max_tokens=1024,\n    messages=[{"role": "user", "content": "Hello, Longdao AI"}]\n)\n\nprint(message.content[0].text)`
  },
  {
    id: 'curl',
    label: 'cURL',
    code: `curl "${normalizedBaseUrl.value}/chat/completions" \\\n  -H "Authorization: Bearer YOUR_LONGDAO_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{\n    "model": "gpt-4o-mini",\n    "messages": [{"role": "user", "content": "Hello, Longdao AI"}]\n  }'`
  }
])

const activeExample = computed(() => examples.value.find((item) => item.id === activeTab.value) ?? examples.value[0])
const safetyItems = computed(() => [
  { title: t('longdao.apiGuide.serverOnly'), description: t('longdao.apiGuide.serverOnlyDesc') },
  { title: t('longdao.apiGuide.rotateKeys'), description: t('longdao.apiGuide.rotateKeysDesc') },
  { title: t('longdao.apiGuide.leastPrivilege'), description: t('longdao.apiGuide.leastPrivilegeDesc') }
])

function copy(value: string) {
  return copyToClipboard(value, t('longdao.apiGuide.copied'))
}
</script>
