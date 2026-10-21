<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.testAccountConnection')"
    width="normal"
    @close="handleClose"
  >
    <div class="space-y-4">
      <!-- Account Info Card -->
      <div
        v-if="account"
        class="flex items-center justify-between rounded-xl border border-gray-200 bg-gradient-to-r from-gray-50 to-gray-100 p-3 dark:border-dark-500 dark:from-dark-700 dark:to-dark-600"
      >
        <div class="flex items-center gap-3">
          <div
            class="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-primary-500 to-primary-600"
          >
            <Icon name="play" size="md" class="text-white" :stroke-width="2" />
          </div>
          <div>
            <div class="font-semibold text-gray-900 dark:text-gray-100">{{ account.name }}</div>
            <div class="flex items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
              <span
                class="rounded bg-gray-200 px-1.5 py-0.5 text-[10px] font-medium uppercase dark:bg-dark-500"
              >
                {{ account.type }}
              </span>
              <span>{{ t('admin.accounts.account') }}</span>
            </div>
          </div>
        </div>
        <span
          :class="[
            'rounded-full px-2.5 py-1 text-xs font-semibold',
            account.status === 'active'
              ? 'bg-green-100 text-green-700 dark:bg-green-500/20 dark:text-green-400'
              : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
          ]"
        >
          {{ account.status }}
        </span>
      </div>

      <div class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.selectTestModel') }}
        </label>
        <Select
          v-model="selectedModelId"
          :options="availableModels"
          :disabled="loadingModels || status === 'connecting'"
          value-key="id"
          label-key="display_name"
          :placeholder="loadingModels ? t('common.loading') + '...' : t('admin.accounts.selectTestModel')"
        />
      </div>

      <div v-if="isOpenAIAccount" class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.openai.testMode') }}
        </label>
        <Select
          v-model="testMode"
          :options="openAITestModeOptions"
          :disabled="status === 'connecting'"
        />
      </div>

      <div v-if="isVideoTest" class="space-y-3" data-testid="video-test-options">
        <TextArea
          v-model="testPrompt"
          :label="t('admin.accounts.openai.videoTestPrompt')"
          :placeholder="t('admin.accounts.openai.videoTestPromptDefault')"
          :disabled="status === 'connecting'"
          rows="3"
        />
        <div class="grid grid-cols-2 gap-3">
          <label class="space-y-1 text-sm text-gray-700 dark:text-gray-300">
            <span>{{ t('admin.accounts.openai.videoTestSeconds') }}</span>
            <input v-model.number="videoSeconds" type="number" min="1" step="1" class="input" :disabled="status === 'connecting'" data-testid="video-test-seconds" />
          </label>
          <label class="space-y-1 text-sm text-gray-700 dark:text-gray-300">
            <span>{{ t('admin.accounts.openai.videoTestSize') }}</span>
            <input v-model="videoSize" type="text" class="input" placeholder="1280x720" :disabled="status === 'connecting'" data-testid="video-test-size" />
          </label>
        </div>
        <p class="rounded-lg bg-amber-50 p-3 text-xs text-amber-800 dark:bg-amber-900/20 dark:text-amber-200">
          {{ t('admin.accounts.openai.videoTestHint') }}
        </p>
        <div class="flex flex-wrap items-end gap-2">
          <label class="min-w-0 flex-1 space-y-1 text-sm text-gray-700 dark:text-gray-300">
            <span>{{ t('admin.accounts.openai.videoTestExistingTask') }}</span>
            <input v-model="videoTaskIDInput" type="text" maxlength="256" class="input" :placeholder="t('admin.accounts.openai.videoTestTaskID')" :disabled="status === 'connecting' || videoTaskLoading" data-testid="video-test-existing-id" />
          </label>
          <button type="button" class="btn btn-secondary rounded-lg px-3 py-2" :disabled="!validExistingVideoTask || status === 'connecting' || videoTaskLoading" data-testid="video-test-lookup" @click="lookupVideoTask">
            <Icon name="refresh" size="sm" />
            {{ t('admin.accounts.openai.videoTestLookup') }}
          </button>
        </div>
      </div>

      <div v-if="supportsImageTest" class="space-y-1.5">
        <TextArea
          v-model="testPrompt"
          :label="t('admin.accounts.imagePromptLabel')"
          :placeholder="t('admin.accounts.imagePromptPlaceholder')"
          :hint="t('admin.accounts.imageTestHint')"
          :disabled="status === 'connecting'"
          rows="3"
        />
      </div>

      <!-- Terminal Output -->
      <div class="group relative">
        <div
          ref="terminalRef"
          class="max-h-[240px] min-h-[120px] overflow-y-auto break-words rounded-xl border border-gray-700 bg-gray-900 p-4 font-mono text-sm dark:border-gray-800 dark:bg-black"
        >
          <!-- Status Line -->
          <div v-if="status === 'idle'" class="flex items-center gap-2 text-gray-500">
            <Icon name="play" size="sm" :stroke-width="2" />
            <span>{{ t('admin.accounts.readyToTest') }}</span>
          </div>
          <div v-else-if="status === 'connecting'" class="flex items-center gap-2 text-yellow-400">
            <Icon name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
            <span>{{ t('admin.accounts.connectingToApi') }}</span>
          </div>

          <!-- Output Lines -->
          <div v-for="(line, index) in outputLines" :key="index" :class="line.class">
            {{ line.text }}
          </div>

          <!-- Streaming Content -->
          <div v-if="streamingContent" class="text-green-400">
            {{ streamingContent }}<span class="animate-pulse">_</span>
          </div>

          <!-- Result Status -->
          <div
            v-if="status === 'success'"
            class="mt-3 flex items-center gap-2 border-t border-gray-700 pt-3"
            :class="videoTaskFailed ? 'text-red-400' : 'text-green-400'"
          >
            <Icon :name="videoTaskFailed ? 'x' : 'check'" size="sm" :stroke-width="2" />
            <span>{{ isVideoTest ? videoTestResultMessage : t('admin.accounts.testCompleted') }}</span>
          </div>
          <div
            v-else-if="status === 'error'"
            class="mt-3 flex items-center gap-2 border-t border-gray-700 pt-3 text-red-400"
          >
            <Icon name="x" size="sm" :stroke-width="2" />
            <span>{{ errorMessage }}</span>
          </div>
        </div>

        <!-- Copy Button -->
        <button
          v-if="outputLines.length > 0"
          @click="copyOutput"
          class="absolute right-2 top-2 rounded-lg bg-gray-800/80 p-1.5 text-gray-400 opacity-0 transition-all hover:bg-gray-700 hover:text-white group-hover:opacity-100"
          :title="t('admin.accounts.copyOutput')"
        >
          <Icon name="link" size="sm" :stroke-width="2" />
        </button>
      </div>

      <div v-if="videoTask" class="space-y-1 rounded-lg border border-gray-200 p-3 text-sm dark:border-dark-500" data-testid="video-test-task">
        <div class="font-medium">{{ t('admin.accounts.openai.videoTestTaskID') }}</div>
        <div class="select-all break-all font-mono">{{ videoTask.id }}</div>
        <div v-if="videoTask.status">{{ t('admin.accounts.openai.videoTestStatus') }}: {{ videoTask.status }}</div>
        <p class="text-xs text-gray-500">{{ t('admin.accounts.openai.videoTestTaskHint') }}</p>
        <div class="flex flex-wrap gap-2 pt-2">
          <button type="button" class="btn btn-secondary rounded-lg px-3 py-2" :disabled="videoTaskLoading" data-testid="video-test-refresh" @click="refreshVideoTask">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': videoTaskLoading }" />
            {{ videoTaskLoading ? t('admin.accounts.openai.videoTestRefreshing') : t('admin.accounts.openai.videoTestRefresh') }}
          </button>
          <a v-if="videoObjectUrl" :href="videoObjectUrl" :download="`${videoTask.id}.mp4`" class="btn btn-secondary rounded-lg px-3 py-2" data-testid="video-test-download">
            <Icon name="download" size="sm" />
            {{ t('admin.accounts.openai.videoTestDownload') }}
          </a>
        </div>
        <p v-if="videoTask.error" class="break-words text-red-600 dark:text-red-400">{{ videoTask.error }}</p>
        <p v-if="videoPlaybackError" class="break-words text-red-600 dark:text-red-400">{{ videoPlaybackError }}</p>
      </div>

      <video v-if="videoObjectUrl" :src="videoObjectUrl" controls playsinline class="aspect-video max-h-[55vh] w-full rounded-lg bg-black object-contain" data-testid="video-test-player" @error="handleVideoPlaybackError" />

      <div v-if="generatedImages.length > 0" class="space-y-2">
        <div class="text-xs font-medium text-gray-600 dark:text-gray-300">
          {{ t('admin.accounts.imagePreview') }}
        </div>
        <div class="flex flex-wrap justify-center gap-3">
          <div
            v-for="(image, index) in generatedImages"
            :key="`${image.url}-${index}`"
            class="group/img relative cursor-pointer overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm transition hover:border-primary-300 hover:shadow-md dark:border-dark-500 dark:bg-dark-700"
            @click="previewImageUrl = image.url"
          >
            <img :src="image.url" :alt="`test-image-${index + 1}`" class="max-h-[360px] w-full object-contain" />
            <div class="absolute inset-0 flex items-center justify-center bg-black/0 transition-colors group-hover/img:bg-black/20">
              <Icon name="eye" size="lg" class="text-white opacity-0 drop-shadow-lg transition-opacity group-hover/img:opacity-100" :stroke-width="2" />
            </div>
            <div class="border-t border-gray-100 px-3 py-1.5 text-xs text-gray-500 dark:border-dark-500 dark:text-gray-300">
              {{ image.mimeType || 'image/*' }}
            </div>
          </div>
        </div>
      </div>

      <!-- Image Lightbox -->
      <Teleport to="body">
        <Transition name="fade">
          <div
            v-if="previewImageUrl"
            class="fixed inset-0 z-[100] flex items-center justify-center bg-black/80 p-4"
            @click.self="previewImageUrl = ''"
          >
            <button
              class="absolute right-4 top-4 rounded-full bg-black/50 p-2 text-white transition-colors hover:bg-black/70"
              @click="previewImageUrl = ''"
            >
              <Icon name="x" size="lg" :stroke-width="2" />
            </button>
            <img
              :src="previewImageUrl"
              alt="preview"
              class="max-h-[90vh] max-w-[90vw] rounded-lg object-contain shadow-2xl"
            />
          </div>
        </Transition>
      </Teleport>

      <!-- Test Info -->
      <div class="flex items-center justify-between px-1 text-xs text-gray-500 dark:text-gray-400">
        <div class="flex items-center gap-3">
          <span class="flex items-center gap-1">
            <Icon name="grid" size="sm" :stroke-width="2" />
            {{ t('admin.accounts.testModel') }}
          </span>
        </div>
        <span class="flex items-center gap-1">
          <Icon name="chat" size="sm" :stroke-width="2" />
          {{
            isVideoTest
              ? t('admin.accounts.openai.testModeVideo')
              : supportsImageTest
              ? t('admin.accounts.imageTestMode')
              : t('admin.accounts.testPrompt')
          }}
        </span>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button
          @click="handleClose"
          class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
        >
          {{ t('common.close') }}
        </button>
        <button
          @click="startTest"
          :disabled="status === 'connecting' || !selectedModelId || !validVideoOptions"
          :class="[
            'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            status === 'connecting' || !selectedModelId || !validVideoOptions
              ? 'cursor-not-allowed bg-primary-400 text-white'
              : status === 'success'
                ? 'bg-green-500 text-white hover:bg-green-600'
                : status === 'error'
                  ? 'bg-orange-500 text-white hover:bg-orange-600'
                  : 'bg-primary-500 text-white hover:bg-primary-600'
          ]"
        >
          <Icon
            v-if="status === 'connecting'"
            name="refresh"
            size="sm"
            class="animate-spin"
            :stroke-width="2"
          />
          <Icon v-else-if="status === 'idle'" name="play" size="sm" :stroke-width="2" />
          <Icon v-else name="refresh" size="sm" :stroke-width="2" />
          <span>
            {{
              status === 'connecting'
                ? t('admin.accounts.testing')
                : status === 'idle'
                  ? (isVideoTest ? t('admin.accounts.openai.videoTestSubmit') : t('admin.accounts.startTest'))
                  : (isVideoTest ? t('admin.accounts.openai.videoTestResubmit') : t('admin.accounts.retry'))
            }}
          </span>
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import TextArea from '@/components/common/TextArea.vue'
import { Icon } from '@/components/icons'
import { useClipboard } from '@/composables/useClipboard'
import { buildApiUrl } from '@/api/client'
import { ADMIN_UI_REQUEST_HEADER } from '@/api/adminUIRequest'
import { adminAPI } from '@/api/admin'
import type { Account, ClaudeModel } from '@/types'

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

interface OutputLine {
  text: string
  class: string
}

interface PreviewImage {
  url: string
  mimeType?: string
}

const props = defineProps<{
  show: boolean
  account: Account | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const terminalRef = ref<HTMLElement | null>(null)
const status = ref<'idle' | 'connecting' | 'success' | 'error'>('idle')
const outputLines = ref<OutputLine[]>([])
const streamingContent = ref('')
const errorMessage = ref('')
const availableModels = ref<ClaudeModel[]>([])
const selectedModelId = ref('')
const testPrompt = ref('')
const loadingModels = ref(false)
let abortController: AbortController | null = null
const generatedImages = ref<PreviewImage[]>([])
const testMode = ref<'default' | 'compact' | 'video'>('default')
const videoSeconds = ref(6)
const videoSize = ref('1280x720')
const videoTaskIDInput = ref('')
const validExistingVideoTask = computed(() => /^[A-Za-z0-9_.-]{1,256}$/.test(videoTaskIDInput.value.trim()) && !['.', '..'].includes(videoTaskIDInput.value.trim()))
const videoTask = ref<{ id: string; status?: string; error?: string; createdAt?: number } | null>(null)
const videoTaskLoading = ref(false)
const videoPlaybackError = ref('')
const videoObjectUrl = ref('')
let videoRequestController: AbortController | null = null
const videoTaskStorageKey = computed(() => props.account ? `sub2api:admin:video-test:v1:${props.account.id}` : '')
const isOpenAIAccount = computed(() => props.account?.platform === 'openai')
const accountSupportsCapability = (capability: string): boolean => {
  const raw = props.account?.credentials?.openai_capabilities
  if (raw == null) return true
  if (Array.isArray(raw)) return raw.includes(capability)
  return typeof raw === 'object' && (raw as Record<string, unknown>)[capability] === true
}
const supportsVideoTest = computed(() => isOpenAIAccount.value && props.account?.type === 'apikey' && accountSupportsCapability('videos'))
const isVideoOnlyAccount = computed(() => supportsVideoTest.value && !accountSupportsCapability('chat_completions'))
const isVideoTest = computed(() => supportsVideoTest.value && testMode.value === 'video')
const videoTaskFailed = computed(() => ['failed', 'error', 'cancelled', 'canceled', 'expired'].includes(videoTask.value?.status?.toLowerCase() || ''))
const videoTestResultMessage = computed(() => {
  if (videoTaskFailed.value) return t('admin.accounts.openai.videoTestFailed')
  if (videoObjectUrl.value) return t('admin.accounts.openai.videoTestReady')
  if (['completed', 'succeeded', 'success', 'finished'].includes(videoTask.value?.status?.toLowerCase() || '')) {
    return t('admin.accounts.openai.videoTestGenerated')
  }
  return t('admin.accounts.openai.videoTestAccepted')
})
const validVideoOptions = computed(() => !isVideoTest.value || (
  Number.isInteger(videoSeconds.value) && videoSeconds.value > 0 && /^[1-9][0-9]{1,4}x[1-9][0-9]{1,4}$/.test(videoSize.value.trim())
))
const openAITestModeOptions = computed(() => [
  ...(!isVideoOnlyAccount.value ? [
    { value: 'default', label: t('admin.accounts.openai.testModeDefault') },
    { value: 'compact', label: t('admin.accounts.openai.testModeCompact') }
  ] : []),
  ...(supportsVideoTest.value ? [{ value: 'video', label: t('admin.accounts.openai.testModeVideo') }] : [])
])
const previewImageUrl = ref('')
const prioritizedGeminiModels = ['gemini-3.1-flash-image', 'gemini-2.5-flash-image', 'gemini-3.5-flash', 'gemini-2.5-flash', 'gemini-2.5-pro', 'gemini-3-flash-preview', 'gemini-3-pro-preview', 'gemini-2.0-flash']
const supportsGeminiImageTest = computed(() => {
  const modelID = selectedModelId.value.toLowerCase()
  if (!modelID.startsWith('gemini-') || !modelID.includes('-image')) return false

  return props.account?.platform === 'gemini' || (props.account?.platform === 'antigravity' && props.account?.type === 'apikey')
})

const supportsOpenAIImageTest = computed(() => {
  const modelID = selectedModelId.value.toLowerCase()
  if (!modelID.startsWith('gpt-image-')) return false
  return props.account?.platform === 'openai'
})

const supportsImageTest = computed(() => !isVideoTest.value && (supportsGeminiImageTest.value || supportsOpenAIImageTest.value))

const sortTestModels = (models: ClaudeModel[]) => {
  const priorityMap = new Map(prioritizedGeminiModels.map((id, index) => [id, index]))

  return [...models].sort((a, b) => {
    const aPriority = priorityMap.get(a.id) ?? Number.MAX_SAFE_INTEGER
    const bPriority = priorityMap.get(b.id) ?? Number.MAX_SAFE_INTEGER
    if (aPriority !== bPriority) return aPriority - bPriority
    return 0
  })
}

watch(selectedModelId, () => {
  if (supportsImageTest.value && !testPrompt.value.trim()) {
    testPrompt.value = t('admin.accounts.imagePromptDefault')
  }
})

let modelLoadVersion = 0
const loadAvailableModels = async () => {
  if (!props.account) return
  const account = props.account
  const version = ++modelLoadVersion
  loadingModels.value = true
  selectedModelId.value = '' // Reset selection before loading
  try {
    const models = await adminAPI.accounts.getAvailableModels(account.id)
    if (version !== modelLoadVersion || !props.show || props.account?.id !== account.id) return
    availableModels.value = props.account.platform === 'gemini' || props.account.platform === 'antigravity'
      ? sortTestModels(models)
      : models
    // Default selection by platform
    if (availableModels.value.length > 0) {
      if (props.account.platform === 'gemini') {
        selectedModelId.value = availableModels.value[0].id
      } else {
        // Try to select Sonnet as default, otherwise use first model
        const sonnetModel = availableModels.value.find((m) => m.id.includes('sonnet'))
        selectedModelId.value = sonnetModel?.id || availableModels.value[0].id
      }
    }
  } catch (error) {
    if (version !== modelLoadVersion || !props.show || props.account?.id !== account.id) return
    console.error('Failed to load available models:', error)
    // Fallback to empty list
    availableModels.value = []
    selectedModelId.value = ''
  } finally {
    if (version === modelLoadVersion) loadingModels.value = false
  }
}

const resetState = () => {
  cancelVideoRequest()
  setVideoObjectUrl('')
  videoPlaybackError.value = ''
  status.value = 'idle'
  outputLines.value = []
  streamingContent.value = ''
  errorMessage.value = ''
  generatedImages.value = []
  previewImageUrl.value = ''
  videoTask.value = null
}

watch([testMode, selectedModelId], () => {
  if (status.value !== 'connecting') resetState()
})

const handleClose = () => {
  abortStream()
  cancelVideoRequest()
  setVideoObjectUrl('')
  emit('close')
}

const setVideoObjectUrl = (url: string) => {
  if (videoObjectUrl.value) URL.revokeObjectURL(videoObjectUrl.value)
  videoObjectUrl.value = url
}

const handleVideoPlaybackError = () => {
  videoPlaybackError.value = t('admin.accounts.openai.videoTestPlaybackFailed')
  setVideoObjectUrl('')
}

const cancelVideoRequest = () => {
  videoRequestController?.abort()
  videoRequestController = null
  videoTaskLoading.value = false
}

const persistVideoTask = () => {
  if (!videoTaskStorageKey.value || !videoTask.value) return
  try {
    sessionStorage.setItem(videoTaskStorageKey.value, JSON.stringify({ accountId: props.account?.id, ...videoTask.value }))
  } catch {
    // The current task remains usable when browser storage is unavailable.
  }
}

const restoreVideoTask = () => {
  if (!supportsVideoTest.value || !videoTaskStorageKey.value) return
  try {
    const raw = sessionStorage.getItem(videoTaskStorageKey.value)
    if (!raw) return
    const saved = JSON.parse(raw)
    if (!saved || saved.accountId !== props.account?.id || typeof saved.id !== 'string' ||
        !/^[A-Za-z0-9_.-]{1,256}$/.test(saved.id) || ['.', '..'].includes(saved.id)) return
    testMode.value = 'video'
    // Apply the mode watch before restoring its task.
    return nextTick().then(() => {
      if (!props.show || saved.accountId !== props.account?.id) return
      videoTask.value = {
        id: saved.id,
        status: typeof saved.status === 'string' ? saved.status : '',
        error: typeof saved.error === 'string' ? saved.error : undefined,
        createdAt: saved.createdAt
      }
      status.value = 'success'
    })
  } catch {
    // Ignore unavailable storage and malformed records.
  }
}

const refreshVideoTask = async () => {
  if (!props.account || !videoTask.value || videoTaskLoading.value) return
  const accountID = props.account.id
  const taskID = videoTask.value.id
  const controller = new AbortController()
  videoRequestController = controller
  videoTaskLoading.value = true
  videoPlaybackError.value = ''
  const isCurrent = () => videoRequestController === controller && !controller.signal.aborted &&
    props.show && props.account?.id === accountID && videoTask.value?.id === taskID
  try {
    const task = await adminAPI.accounts.getVideoTestTask(accountID, taskID, controller.signal)
    if (!isCurrent()) return
    status.value = 'success'
    videoTask.value = { ...videoTask.value!, status: task.status, error: task.error }
    persistVideoTask()
    if (['completed', 'succeeded', 'success', 'finished'].includes(task.status.toLowerCase())) {
      if (!videoObjectUrl.value) {
        const content = await adminAPI.accounts.getVideoTestContent(accountID, taskID, controller.signal)
        if (!isCurrent()) return
        setVideoObjectUrl(URL.createObjectURL(content))
      }
    } else {
      setVideoObjectUrl('')
    }
  } catch (error) {
    if (!isCurrent()) return
    videoPlaybackError.value = (error as { message?: string })?.message || t('admin.accounts.openai.videoTestQueryFailed')
  } finally {
    if (videoRequestController === controller) {
      videoRequestController = null
      videoTaskLoading.value = false
    }
  }
}

const lookupVideoTask = async () => {
  if (!validExistingVideoTask.value || !supportsVideoTest.value || status.value === 'connecting' || videoTaskLoading.value) return
  resetState()
  videoTask.value = { id: videoTaskIDInput.value.trim(), createdAt: Date.now() }
  await refreshVideoTask()
}

onBeforeUnmount(() => {
  abortStream()
  cancelVideoRequest()
  setVideoObjectUrl('')
})

const abortStream = () => {
  if (abortController) {
    abortController.abort()
    abortController = null
  }
}

const addLine = (text: string, className: string = 'text-gray-300') => {
  outputLines.value.push({ text, class: className })
  scrollToBottom()
}

const scrollToBottom = async () => {
  await nextTick()
  if (terminalRef.value) {
    terminalRef.value.scrollTop = terminalRef.value.scrollHeight
  }
}

// Load available models when opened, including an initially visible dialog.
let dialogVersion = 0
watch(
  () => [props.show, props.account?.id] as const,
  async ([newVal]) => {
    const version = ++dialogVersion
    abortStream()
    resetState()
    if (newVal && props.account) {
      const accountID = props.account.id
      testPrompt.value = ''
      testMode.value = isVideoOnlyAccount.value ? 'video' : 'default'
      videoSeconds.value = 6
      videoSize.value = '1280x720'
      videoTaskIDInput.value = ''
      await loadAvailableModels()
      await nextTick()
      if (version === dialogVersion && props.show && props.account?.id === accountID) await restoreVideoTask()
    }
  },
  { immediate: true }
)

const startTest = async () => {
  if (status.value === 'connecting' || !props.account || !selectedModelId.value || !validVideoOptions.value) return

  resetState()
  status.value = 'connecting'
  addLine(t('admin.accounts.startingTestForAccount', { name: props.account.name }), 'text-blue-400')
  addLine(t('admin.accounts.testAccountTypeLabel', { type: props.account.type }), 'text-gray-400')
  addLine('', 'text-gray-300')

  abortStream()

  const controller = new AbortController()
  abortController = controller
  const accountID = props.account.id
  const isCurrent = () => abortController === controller && !controller.signal.aborted &&
    props.show && props.account?.id === accountID

  try {
    // Use the configured API base; EventSource does not support POST.
    const url = buildApiUrl(`/admin/accounts/${props.account.id}/test`)

    // Use fetch with streaming for SSE since EventSource doesn't support POST
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('auth_token')}`,
        'Content-Type': 'application/json',
        [ADMIN_UI_REQUEST_HEADER]: '1'
      },
      body: JSON.stringify({
        model_id: selectedModelId.value,
        prompt: supportsImageTest.value || isVideoTest.value ? testPrompt.value.trim() : '',
        ...(isOpenAIAccount.value ? { mode: testMode.value } : {}),
        ...(isVideoTest.value ? { video: { seconds: videoSeconds.value, size: videoSize.value.trim() } } : {})
      }),
      signal: controller.signal
    })

    if (!isCurrent()) return

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new Error('No response body')
    }

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (!isCurrent()) { await reader.cancel(); return }
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const jsonStr = line.slice(6).trim()
          if (jsonStr) {
            try {
              const event = JSON.parse(jsonStr)
              handleEvent(event)
            } catch (e) {
              console.error('Failed to parse SSE event:', e)
            }
          }
        }
      }
    }
    if (status.value === 'connecting') {
      throw new Error(isVideoTest.value
        ? t('admin.accounts.openai.videoTestInterrupted')
        : t('admin.accounts.testFailed'))
    }
  } catch (error: unknown) {
    if (!isCurrent()) return
    if (error instanceof DOMException && error.name === 'AbortError') {
      status.value = 'idle'
      return
    }
    status.value = 'error'
    const msg = error instanceof Error ? error.message : 'Unknown error'
    errorMessage.value = msg
    addLine(`Error: ${msg}`, 'text-red-400')
  }
}

const handleEvent = (event: {
  type: string
  text?: string
  model?: string
  success?: boolean
  error?: string
  image_url?: string
  mime_type?: string
  task_id?: string
  status?: string
}) => {
  switch (event.type) {
    case 'test_start':
      addLine(t('admin.accounts.connectedToApi'), 'text-green-400')
      if (event.model) {
        addLine(t('admin.accounts.usingModel', { model: event.model }), 'text-cyan-400')
      }
      addLine(
        isVideoTest.value
          ? t('admin.accounts.openai.videoTestSending')
          : supportsImageTest.value
            ? t('admin.accounts.sendingImageRequest')
            : t('admin.accounts.sendingTestMessage'),
        'text-gray-400'
      )
      addLine('', 'text-gray-300')
      addLine(t('admin.accounts.response'), 'text-yellow-400')
      break

    case 'content':
      if (event.text) {
        streamingContent.value += event.text
        scrollToBottom()
      }
      break

    case 'status':
      if (event.text) {
        addLine(event.text, 'text-cyan-300')
      }
      break

    case 'image':
      if (event.image_url) {
        generatedImages.value.push({
          url: event.image_url,
          mimeType: event.mime_type
        })
        addLine(t('admin.accounts.imageReceived', { count: generatedImages.value.length }), 'text-purple-300')
      }
      break

    case 'video_task':
      if (event.task_id) {
        videoTask.value = { id: event.task_id, status: event.status, createdAt: Date.now() }
        persistVideoTask()
        addLine(`${t('admin.accounts.openai.videoTestTaskID')}: ${event.task_id}`, 'text-purple-300')
      }
      break

    case 'test_complete':
      // Move streaming content to output lines
      if (streamingContent.value) {
        addLine(streamingContent.value, 'text-green-300')
        streamingContent.value = ''
      }
      if (event.success) {
        status.value = 'success'
      } else {
        status.value = 'error'
        errorMessage.value = event.error || 'Test failed'
      }
      break

    case 'error':
      status.value = 'error'
      errorMessage.value = event.error || 'Unknown error'
      if (streamingContent.value) {
        addLine(streamingContent.value, 'text-green-300')
        streamingContent.value = ''
      }
      break
  }
}

const copyOutput = () => {
  const text = outputLines.value.map((l) => l.text).join('\n')
  copyToClipboard(text, t('admin.accounts.outputCopied'))
}
</script>

<style>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
