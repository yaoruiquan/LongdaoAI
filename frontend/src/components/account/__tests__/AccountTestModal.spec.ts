import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import AccountTestModal from '../AccountTestModal.vue'
import { ADMIN_UI_REQUEST_HEADER } from '@/api/adminUIRequest'

const { getAvailableModelsMock, getVideoTestTaskMock, getVideoTestContentMock } = vi.hoisted(() => ({
  getAvailableModelsMock: vi.fn(),
  getVideoTestTaskMock: vi.fn(),
  getVideoTestContentMock: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels: getAvailableModelsMock,
      getVideoTestTask: getVideoTestTaskMock,
      getVideoTestContent: getVideoTestContentMock
    }
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: { type: Boolean, default: false } },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: {
    modelValue: { type: [String, Number, Boolean, null], default: '' },
    options: { type: Array, default: () => [] },
    valueKey: { type: String, default: 'value' },
    labelKey: { type: String, default: 'label' }
  },
  emits: ['update:modelValue'],
  template: `
    <select
      v-bind="$attrs"
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option
        v-for="option in options"
        :key="option[valueKey]"
        :value="option[valueKey]"
      >
        {{ option[labelKey] }}
      </option>
    </select>
  `
})

const TextAreaStub = defineComponent({
  name: 'TextArea',
  props: {
    modelValue: { type: String, default: '' }
  },
  emits: ['update:modelValue'],
  template: `
    <textarea
      v-bind="$attrs"
      :value="modelValue"
      @input="$emit('update:modelValue', $event.target.value)"
    />
  `
})

function buildAccount() {
  return {
    id: 1,
    name: 'OpenAI OAuth',
    platform: 'openai',
    type: 'oauth',
    status: 'active',
    credentials: {},
    extra: {},
    concurrency: 1,
    priority: 1,
    proxy_id: null,
    auto_pause_on_expired: false
  } as any
}

describe('AccountTestModal', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    sessionStorage.clear()
    getVideoTestTaskMock.mockReset()
    getVideoTestContentMock.mockReset()
    vi.stubGlobal('URL', Object.assign(class extends URL {}, {
      createObjectURL: vi.fn(() => 'blob:video-test'),
      revokeObjectURL: vi.fn()
    }))
    getAvailableModelsMock.mockReset()
    getAvailableModelsMock.mockResolvedValue([
      { id: 'gpt-5.4', display_name: 'GPT-5.4' }
    ])
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => ({
          read: vi.fn().mockResolvedValue({ done: true, value: undefined })
        })
      }
    } as any)
    localStorage.setItem('auth_token', 'test-token')
  })

  afterEach(() => {
    global.fetch = originalFetch
    localStorage.clear()
    sessionStorage.clear()
    vi.unstubAllGlobals()
  })

  it('posts compact mode for OpenAI compact probe', async () => {
    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: buildAccount()
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    ;(wrapper.vm as any).testMode = 'compact'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, options] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(options.body)).toMatchObject({
      model_id: 'gpt-5.4',
      mode: 'compact'
    })
  })

  it('renders Chat Completions path status from test SSE', async () => {
    const encoder = new TextEncoder()
    const chunks = [
      encoder.encode('data: {"type":"status","text":"已通过 /v1/chat/completions 验证"}\n\n'),
      encoder.encode('data: {"type":"test_complete","success":true}\n\n')
    ]
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: {
        getReader: () => ({
          read: vi.fn().mockImplementation(() => Promise.resolve(
            chunks.length > 0
              ? { done: false, value: chunks.shift() }
              : { done: true, value: undefined }
          ))
        })
      }
    } as any)

    const wrapper = mount(AccountTestModal, {
      props: {
        show: true,
        account: buildAccount()
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          TextArea: TextAreaStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).selectedModelId = 'gpt-5.4'
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(wrapper.text()).toContain('已通过 /v1/chat/completions 验证')
  })

  it.each([['videos'], { videos: true }])('defaults a video-only account to video submission: %j', async (capabilities) => {
    getAvailableModelsMock.mockResolvedValue([{ id: 'minimax-h3', display_name: 'MiniMax H3' }])
    const chunks = [new TextEncoder().encode(
      'data: {"type":"video_task","task_id":"task_123","status":"queued"}\n\n' +
      'data: {"type":"test_complete","success":true}\n\n'
    )]
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      body: { getReader: () => ({ read: vi.fn().mockImplementation(() => Promise.resolve(
        chunks.length ? { done: false, value: chunks.shift() } : { done: true }
      )) }) }
    } as any)
    const account = { ...buildAccount(), type: 'apikey', credentials: { openai_capabilities: capabilities } }
    const wrapper = mount(AccountTestModal, {
      props: { show: true, account },
      global: { stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true } }
    })
    await flushPromises()
    expect(getAvailableModelsMock).toHaveBeenCalledWith(account.id)
    expect(wrapper.find('[data-testid="video-test-options"]').exists()).toBe(true)
    const modeOptions = wrapper.findAll('select')[1].findAll('option')
    expect(modeOptions.map((o) => o.attributes('value'))).toEqual(['video'])
    await wrapper.find('textarea').setValue('paper boat')
    await wrapper.find('[data-testid="video-test-seconds"]').setValue('10')
    await wrapper.find('[data-testid="video-test-size"]').setValue('1920x1080')
    await (wrapper.vm as any).startTest()
    await flushPromises()
    expect(global.fetch).toHaveBeenCalledTimes(1)
    const [, options] = (global.fetch as any).mock.calls[0]
    expect(JSON.parse(options.body)).toEqual({
      model_id: 'minimax-h3', mode: 'video', prompt: 'paper boat', video: { seconds: 10, size: '1920x1080' }
    })
    expect(options.headers[ADMIN_UI_REQUEST_HEADER]).toBe('1')
    expect(wrapper.get('[data-testid="video-test-task"]').text()).toContain('task_123')
    expect(wrapper.get('[data-testid="video-test-task"]').text()).toContain('queued')
    expect(wrapper.text()).toContain('admin.accounts.openai.videoTestAccepted')
    expect(wrapper.text()).not.toContain('admin.accounts.testCompleted')
    wrapper.unmount()
  })

  it('lets mixed accounts explicitly choose video without changing their default text mode', async () => {
    const wrapper = mount(AccountTestModal, {
      props: { show: true, account: { ...buildAccount(), type: 'apikey' } },
      global: { stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true } }
    })
    await flushPromises()
    expect(wrapper.find('[data-testid="video-test-options"]').exists()).toBe(false)
    await wrapper.findAll('select')[1].setValue('video')
    expect(wrapper.find('[data-testid="video-test-options"]').exists()).toBe(true)
    await wrapper.get('[data-testid="video-test-seconds"]').setValue('-1')
    await (wrapper.vm as any).startTest()
    expect(global.fetch).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('reports an interrupted video submission without success or automatic retries', async () => {
    const wrapper = mount(AccountTestModal, {
      props: { show: true, account: { ...buildAccount(), type: 'apikey', credentials: { openai_capabilities: ['videos'] } } },
      global: { stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true } }
    })
    await flushPromises()
    await (wrapper.vm as any).startTest()
    await flushPromises()
    expect(global.fetch).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('admin.accounts.openai.videoTestInterrupted')
    expect(wrapper.text()).not.toContain('admin.accounts.openai.videoTestAccepted')
    wrapper.unmount()
  })

  it('prevents concurrent video submissions and clears results when changing mode', async () => {
    let finishRequest!: (response: Response) => void
    global.fetch = vi.fn().mockImplementation(() => new Promise<Response>((resolve) => {
      finishRequest = resolve
    }))
    const wrapper = mount(AccountTestModal, {
      props: { show: true, account: { ...buildAccount(), type: 'apikey' } },
      global: { stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true } }
    })
    await flushPromises()
    await wrapper.findAll('select')[1].setValue('video')
    const firstRequest = (wrapper.vm as any).startTest()
    await (wrapper.vm as any).startTest()
    expect(global.fetch).toHaveBeenCalledTimes(1)
    finishRequest(new Response(
      'data: {"type":"video_task","task_id":"task_123","status":"queued"}\n\n' +
      'data: {"type":"test_complete","success":true}\n\n'
    ))
    await firstRequest
    await flushPromises()
    expect(wrapper.find('[data-testid="video-test-task"]').exists()).toBe(true)
    await wrapper.findAll('select')[1].setValue('default')
    expect(wrapper.find('[data-testid="video-test-task"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('admin.accounts.testCompleted')
    wrapper.unmount()
  })

  const mountSavedTask = async (taskStatus = 'queued') => {
    sessionStorage.setItem('sub2api:admin:video-test:v1:1', JSON.stringify({
      accountId: 1, id: 'task_saved', status: taskStatus, createdAt: Date.now()
    }))
    const wrapper = mount(AccountTestModal, {
      props: { show: true, account: { ...buildAccount(), type: 'apikey' } },
      global: { stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true } }
    })
    await flushPromises()
    return wrapper
  }

  it('restores a previous task without submitting or fetching anything', async () => {
    const wrapper = await mountSavedTask()
    expect(wrapper.get('[data-testid="video-test-task"]').text()).toContain('task_saved')
    expect(wrapper.get('[data-testid="video-test-task"]').text()).toContain('queued')
    expect(global.fetch).not.toHaveBeenCalled()
    expect(getVideoTestTaskMock).not.toHaveBeenCalled()
    expect(getVideoTestContentMock).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('looks up an older task ID without a paid submission', async () => {
    const wrapper = await mountSavedTask()
    getVideoTestTaskMock.mockResolvedValue({ id: 'task_before_upgrade', status: 'processing' })
    await wrapper.get('[data-testid="video-test-existing-id"]').setValue('task_before_upgrade')
    await wrapper.get('[data-testid="video-test-lookup"]').trigger('click')
    await flushPromises()
    expect(getVideoTestTaskMock).toHaveBeenCalledWith(1, 'task_before_upgrade', expect.any(AbortSignal))
    expect(wrapper.get('[data-testid="video-test-task"]').text()).toContain('task_before_upgrade')
    expect(sessionStorage.getItem('sub2api:admin:video-test:v1:1')).toContain('task_before_upgrade')
    expect(global.fetch).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('rejects unsafe manual task IDs and retains the saved task when lookup fails', async () => {
    const wrapper = await mountSavedTask()
    await wrapper.get('[data-testid="video-test-existing-id"]').setValue('../invalid')
    expect(wrapper.get('[data-testid="video-test-lookup"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-testid="video-test-existing-id"]').setValue('task_missing')
    getVideoTestTaskMock.mockRejectedValue({ message: 'Task not found' })
    await wrapper.get('[data-testid="video-test-lookup"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Task not found')
    expect(sessionStorage.getItem('sub2api:admin:video-test:v1:1')).toContain('task_saved')
    expect(global.fetch).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it.each(['queued', 'processing', 'unknown-provider-status'])('refreshes %s without fetching content or creating tasks', async (taskStatus) => {
    const wrapper = await mountSavedTask()
    getVideoTestTaskMock.mockResolvedValue({ id: 'task_saved', status: taskStatus })
    await wrapper.get('[data-testid="video-test-refresh"]').trigger('click')
    await flushPromises()
    expect(getVideoTestTaskMock).toHaveBeenCalledWith(1, 'task_saved', expect.any(AbortSignal))
    expect(wrapper.get('[data-testid="video-test-task"]').text()).toContain(taskStatus)
    expect(getVideoTestContentMock).not.toHaveBeenCalled()
    expect(wrapper.find('video').exists()).toBe(false)
    expect(global.fetch).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('loads a completed task into a player and download link, reusing the loaded video', async () => {
    const wrapper = await mountSavedTask()
    getVideoTestTaskMock.mockResolvedValue({ id: 'task_saved', status: 'completed' })
    const blob = new Blob(['mp4-test'], { type: 'video/mp4' })
    getVideoTestContentMock.mockResolvedValue(blob)
    await wrapper.get('[data-testid="video-test-refresh"]').trigger('click')
    await flushPromises()
    expect(getVideoTestContentMock).toHaveBeenCalledWith(1, 'task_saved', expect.any(AbortSignal))
    expect(URL.createObjectURL).toHaveBeenCalledWith(blob)
    expect(wrapper.get('video').attributes('src')).toBe('blob:video-test')
    expect(wrapper.get('video').attributes('controls')).toBeDefined()
    expect(wrapper.get('[data-testid="video-test-download"]').attributes('download')).toBe('task_saved.mp4')
    expect(wrapper.text()).toContain('admin.accounts.openai.videoTestReady')
    await wrapper.get('[data-testid="video-test-refresh"]').trigger('click')
    await flushPromises()
    expect(getVideoTestContentMock).toHaveBeenCalledTimes(1)
    expect(global.fetch).not.toHaveBeenCalled()
    wrapper.unmount()
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:video-test')
    expect(sessionStorage.getItem('sub2api:admin:video-test:v1:1')).toContain('task_saved')
  })

  it('shows a terminal failure and its reason without fetching content', async () => {
    const wrapper = await mountSavedTask()
    getVideoTestTaskMock.mockResolvedValue({ id: 'task_saved', status: 'failed', error: 'Generation rejected' })
    await wrapper.get('[data-testid="video-test-refresh"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Generation rejected')
    expect(getVideoTestContentMock).not.toHaveBeenCalled()
    expect(wrapper.find('video').exists()).toBe(false)
    wrapper.unmount()

    const restored = mount(AccountTestModal, {
      props: { show: true, account: { ...buildAccount(), type: 'apikey' } },
      global: { stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true } }
    })
    await flushPromises()
    expect(restored.get('[data-testid="video-test-task"]').text()).toContain('Generation rejected')
    expect(restored.text()).toContain('admin.accounts.openai.videoTestFailed')
    expect(getVideoTestTaskMock).toHaveBeenCalledTimes(1)
    expect(global.fetch).not.toHaveBeenCalled()
    restored.unmount()
  })

  it('keeps the original task after a query error and allows a safe refresh retry', async () => {
    const wrapper = await mountSavedTask()
    getVideoTestTaskMock.mockRejectedValueOnce({ message: 'Task temporarily unavailable' })
      .mockResolvedValue({ id: 'task_saved', status: 'processing' })
    await wrapper.get('[data-testid="video-test-refresh"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Task temporarily unavailable')
    expect(wrapper.text()).toContain('task_saved')
    await wrapper.get('[data-testid="video-test-refresh"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('processing')
    expect(wrapper.text()).not.toContain('Task temporarily unavailable')
    expect(global.fetch).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('allows retrying a failed video download without submitting a new task', async () => {
    const wrapper = await mountSavedTask()
    getVideoTestTaskMock.mockResolvedValue({ id: 'task_saved', status: 'completed' })
    getVideoTestContentMock.mockRejectedValueOnce({ message: 'Video unavailable' })
      .mockResolvedValue(new Blob(['video'], { type: 'video/mp4' }))
    await wrapper.get('[data-testid="video-test-refresh"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Video unavailable')
    expect(wrapper.find('video').exists()).toBe(false)
    await wrapper.get('[data-testid="video-test-refresh"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('video').exists()).toBe(true)
    expect(global.fetch).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('discards an unplayable blob and downloads it again on refresh', async () => {
    const wrapper = await mountSavedTask()
    getVideoTestTaskMock.mockResolvedValue({ id: 'task_saved', status: 'completed' })
    getVideoTestContentMock.mockResolvedValue(new Blob(['video'], { type: 'video/mp4' }))
    vi.mocked(URL.createObjectURL).mockReturnValueOnce('blob:broken').mockReturnValueOnce('blob:retry')
    await wrapper.get('[data-testid="video-test-refresh"]').trigger('click')
    await flushPromises()
    await wrapper.get('video').trigger('error')
    expect(wrapper.text()).toContain('admin.accounts.openai.videoTestPlaybackFailed')
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:broken')
    expect(wrapper.find('video').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('admin.accounts.openai.videoTestReady')

    await wrapper.get('[data-testid="video-test-refresh"]').trigger('click')
    await flushPromises()
    expect(getVideoTestContentMock).toHaveBeenCalledTimes(2)
    expect(wrapper.get('video').attributes('src')).toBe('blob:retry')
    expect(wrapper.text()).not.toContain('admin.accounts.openai.videoTestPlaybackFailed')
    expect(global.fetch).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('ignores late query responses when the selected account changes', async () => {
    const wrapper = await mountSavedTask()
    let resolveQuery!: (value: unknown) => void
    getVideoTestTaskMock.mockImplementation(() => new Promise((resolve) => { resolveQuery = resolve }))
    await wrapper.get('[data-testid="video-test-refresh"]').trigger('click')
    const signal = getVideoTestTaskMock.mock.calls[0][2] as AbortSignal
    await wrapper.setProps({ account: { ...buildAccount(), type: 'apikey', id: 2 } })
    await flushPromises()
    expect(signal.aborted).toBe(true)
    resolveQuery({ id: 'task_saved', status: 'completed' })
    await flushPromises()
    expect(getVideoTestContentMock).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="video-test-task"]').exists()).toBe(false)
    expect(wrapper.find('video').exists()).toBe(false)
    wrapper.unmount()
  })

  it('cancels pending downloads on close and restores the task on reopen', async () => {
    const wrapper = await mountSavedTask()
    getVideoTestTaskMock.mockResolvedValue({ id: 'task_saved', status: 'completed' })
    let finishDownload!: (value: Blob) => void
    getVideoTestContentMock.mockImplementation(() => new Promise((resolve) => { finishDownload = resolve }))
    await wrapper.get('[data-testid="video-test-refresh"]').trigger('click')
    await flushPromises()
    const signal = getVideoTestContentMock.mock.calls[0][2] as AbortSignal
    await wrapper.setProps({ show: false })
    expect(signal.aborted).toBe(true)
    finishDownload(new Blob(['video'], { type: 'video/mp4' }))
    await flushPromises()
    expect(URL.createObjectURL).not.toHaveBeenCalled()
    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(wrapper.text()).toContain('task_saved')
    expect(global.fetch).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it.each(['not-json', JSON.stringify({ accountId: 2, id: 'task_other' }), JSON.stringify({ accountId: 1, id: '../bad' })])('ignores malformed or mismatched stored tasks', async (record) => {
    sessionStorage.setItem('sub2api:admin:video-test:v1:1', record)
    const wrapper = mount(AccountTestModal, {
      props: { show: true, account: { ...buildAccount(), type: 'apikey' } },
      global: { stubs: { BaseDialog: BaseDialogStub, Select: SelectStub, TextArea: TextAreaStub, Icon: true } }
    })
    await flushPromises()
    expect(wrapper.find('[data-testid="video-test-task"]').exists()).toBe(false)
    expect(global.fetch).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
