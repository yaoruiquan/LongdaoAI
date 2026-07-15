<template>
  <div class="relative overflow-hidden rounded-2xl border border-primary-200 bg-primary-50/70 p-4 dark:border-primary-900/70 dark:bg-primary-950/20">
    <div class="absolute inset-y-0 left-0 w-1 bg-primary-500"></div>
    <div class="flex items-start gap-3 pl-1">
      <div class="mt-0.5 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-xl bg-primary-100 text-primary-700 dark:bg-primary-900/50 dark:text-primary-300">
        <slot name="icon">i</slot>
      </div>
      <div>
        <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ title }}</p>
        <p class="mt-1 text-xs leading-5 text-gray-600 dark:text-dark-300">{{ description }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ title: string; description: string }>()
</script>
