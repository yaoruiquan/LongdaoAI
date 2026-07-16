import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ApiIntegrationGuide from '../ApiIntegrationGuide.vue'

const copyToClipboard = vi.fn().mockResolvedValue(true)

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

describe('ApiIntegrationGuide', () => {
  it('renders the configured endpoint and switches between real code examples', async () => {
    const wrapper = mount(ApiIntegrationGuide, {
      props: { baseUrl: 'https://api.longdao.example/v1/' }
    })

    expect(wrapper.text()).toContain('https://api.longdao.example/v1')
    expect(wrapper.text()).toContain('YOUR_LONGDAO_API_KEY')
    expect(wrapper.text()).toContain('from openai import OpenAI')

    await wrapper.get('button:nth-of-type(3)').trigger('click')
    expect(wrapper.text()).toContain('curl "https://api.longdao.example/v1/chat/completions"')

    await wrapper.get('button').trigger('click')
    expect(copyToClipboard).toHaveBeenCalledWith(
      'https://api.longdao.example/v1',
      'longdao.apiGuide.copied'
    )
  })
})
