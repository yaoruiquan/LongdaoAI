<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
          <div class="relative w-full sm:w-80">
            <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model="searchQuery" type="text" class="input pl-10" :placeholder="t('availableModels.searchPlaceholder')" />
          </div>
          <button class="btn btn-secondary" :disabled="loading" :title="t('common.refresh')" @click="loadModels">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </template>

      <template #table>
        <div v-if="loading" class="py-16 text-center">
          <Icon name="refresh" size="xl" class="inline-block animate-spin text-gray-400" />
        </div>
        <div v-else-if="filteredGroups.length === 0" class="py-16 text-center">
          <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-gray-400" />
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('availableModels.empty') }}</p>
        </div>
        <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
          <section v-for="group in filteredGroups" :key="group.id" class="p-5">
            <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
              <div class="flex min-w-0 items-center gap-3">
                <PlatformIcon :platform="group.platform as GroupPlatform" size="sm" />
                <div class="min-w-0">
                  <h2 class="truncate font-medium text-gray-900 dark:text-white">{{ group.name }}</h2>
                  <p class="text-xs text-gray-500 dark:text-gray-400">
                    {{ group.platform }} · {{ t('availableModels.modelCount', { count: group.models.length }) }}
                  </p>
                </div>
              </div>
              <button
                class="btn btn-secondary text-sm"
                :disabled="group.models.length === 0"
                :title="t('availableModels.copyAll')"
                @click="copyGroup(group)"
              >
                <Icon :name="copiedGroupId === group.id ? 'check' : 'clipboard'" size="sm" class="mr-1.5" />
                {{ copiedGroupId === group.id ? t('availableModels.copied') : t('availableModels.copyAll') }}
              </button>
            </div>

            <div v-if="group.models.length" class="flex flex-wrap gap-2">
              <div
                v-for="model in group.models"
                :key="model"
                class="inline-flex items-center gap-1 rounded-md border border-gray-200 bg-gray-50 px-2.5 py-1 font-mono text-xs text-gray-700 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300"
              >
                <span>{{ model }}</span>
                <button
                  class="rounded p-0.5 text-gray-400 transition-colors hover:bg-gray-200 hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-gray-200"
                  :title="t('availableModels.copyModel')"
                  @click="copyModel(model, group.id)"
                >
                  <Icon :name="copiedModelKey === `${group.id}:${model}` ? 'check' : 'clipboard'" size="xs" />
                </button>
              </div>
            </div>
            <p v-else class="text-sm text-gray-400 dark:text-gray-500">{{ t('availableModels.noModels') }}</p>
          </section>
        </div>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type { GroupPlatform } from '@/types'
import userGroupsAPI, { type UserAvailableModelsGroup } from '@/api/groups'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const groups = ref<UserAvailableModelsGroup[]>([])
const searchQuery = ref('')
const loading = ref(false)
const copiedModelKey = ref<string | null>(null)
const copiedGroupId = ref<number | null>(null)

const filteredGroups = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return groups.value
  return groups.value
    .map((group) => {
      if (group.name.toLowerCase().includes(query) || group.platform.toLowerCase().includes(query)) return group
      const models = group.models.filter((model) => model.toLowerCase().includes(query))
      return models.length ? { ...group, models } : null
    })
    .filter((group): group is UserAvailableModelsGroup => group !== null)
})

async function loadModels() {
  loading.value = true
  try {
    groups.value = await userGroupsAPI.getAvailableModels()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('availableModels.loadError')))
  } finally {
    loading.value = false
  }
}

async function copyModel(model: string, groupId: number) {
  if (await copyToClipboard(model, t('availableModels.copied'))) {
    copiedModelKey.value = `${groupId}:${model}`
    window.setTimeout(() => { copiedModelKey.value = null }, 1200)
  }
}

async function copyGroup(group: UserAvailableModelsGroup) {
  if (await copyToClipboard(group.models.join('\n'), t('availableModels.copied'))) {
    copiedGroupId.value = group.id
    window.setTimeout(() => { copiedGroupId.value = null }, 1200)
  }
}

onMounted(loadModels)
</script>
