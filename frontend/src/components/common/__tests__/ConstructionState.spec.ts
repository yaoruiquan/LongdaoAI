import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ConstructionState from '../ConstructionState.vue'

const messages: Record<string, string> = {
  'longdao.adminConstruction.badge': '建设中',
  'longdao.adminConstruction.futureCapabilities': '未来能力',
  'longdao.adminConstruction.statusTitle': '接入状态',
  'longdao.adminConstruction.statusDescription': '后端接口和数据流尚未接入。',
  'longdao.adminConstruction.disabledNotice': '当前页面仅用于导航占位，不提供可执行操作。',
  title: '支付订单管理',
  description: '统一展示待建设的支付订单能力。',
  capabilityA: '订单检索',
  capabilityB: '对账导出',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

describe('ConstructionState', () => {
  it('renders construction status and future capabilities', () => {
    const wrapper = mount(ConstructionState, {
      props: {
        titleKey: 'title',
        descriptionKey: 'description',
        capabilityKeys: ['capabilityA', 'capabilityB'],
      },
    })

    expect(wrapper.text()).toContain('建设中')
    expect(wrapper.text()).toContain('支付订单管理')
    expect(wrapper.text()).toContain('统一展示待建设的支付订单能力。')
    expect(wrapper.text()).toContain('订单检索')
    expect(wrapper.text()).toContain('对账导出')
    expect(wrapper.text()).toContain('当前页面仅用于导航占位，不提供可执行操作。')
  })
})
