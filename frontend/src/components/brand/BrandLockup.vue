<template>
  <div class="inline-flex items-center gap-3" :class="compact ? 'gap-2.5' : 'gap-3'">
    <div
      class="flex shrink-0 items-center justify-center overflow-hidden rounded-2xl bg-white shadow-lg shadow-primary-500/20 ring-1 ring-primary-500/10 dark:bg-dark-900 dark:ring-white/10"
      :class="compact ? 'h-10 w-10' : 'h-12 w-12'"
    >
      <img :src="resolvedLogo" :alt="name" class="h-full w-full object-contain" />
    </div>
    <div v-if="showText" class="min-w-0 text-left">
      <p
        class="truncate font-semibold tracking-tight text-gray-950 dark:text-white"
        :class="compact ? 'text-base' : 'text-xl'"
      >
        {{ name }}
      </p>
      <p
        v-if="subtitle"
        class="truncate text-xs font-medium uppercase tracking-[0.24em] text-primary-600 dark:text-primary-300"
      >
        {{ subtitle }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { sanitizeUrl } from '@/utils/url'

const props = withDefaults(
  defineProps<{
    name?: string
    logo?: string
    subtitle?: string
    compact?: boolean
    showText?: boolean
  }>(),
  {
    name: '龙道 AI',
    logo: '',
    subtitle: '',
    compact: false,
    showText: true
  }
)

const resolvedLogo = computed(
  () => sanitizeUrl(props.logo || '/longdao-logo.png', { allowRelative: true, allowDataUrl: true }) || '/longdao-logo.png'
)
</script>
