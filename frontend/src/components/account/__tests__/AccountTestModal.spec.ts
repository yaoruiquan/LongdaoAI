import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import AccountTestModal from '../AccountTestModal.vue'
import { ADMIN_UI_REQUEST_HEADER } from '@/api/adminUIRequest'

const { getAvailableModelsMock } = vi.hoisted(() => ({
  getAvailableModelsMock: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getAvailableModels: getAvailableModelsMock
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
})
