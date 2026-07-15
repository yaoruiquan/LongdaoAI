import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import RiskEventsView from '../RiskEventsView.vue'

const messages: Record<string, string> = {
  'longdao.adminConstruction.badge': '功能建设中',
  'longdao.adminConstruction.futureCapabilities': '未来能力',
  'longdao.adminConstruction.statusTitle': '接入状态',
  'longdao.adminConstruction.statusDescription': '后台服务尚未接入。',
  'longdao.adminConstruction.disabledNotice': '当前页面不会执行任何管理操作。',
  'longdao.adminConstruction.riskEvents.title': '风控事件',
  'longdao.adminConstruction.riskEvents.description': '用于跟踪可疑支付和账户风险。',
  'longdao.adminConstruction.riskEvents.capabilities.rules': '风险规则',
  'longdao.adminConstruction.riskEvents.capabilities.alerts': '事件告警',
  'longdao.adminConstruction.riskEvents.capabilities.reviewQueue': '人工复核',
  'longdao.adminConstruction.riskEvents.capabilities.reporting': '风险报表',
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

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: { template: '<main><slot /></main>' },
}))

const AppLayoutStub = { template: '<main><slot /></main>' }

describe('RiskEventsView', () => {
  it('renders the shared construction state and future risk event capabilities', () => {
    const wrapper = mount(RiskEventsView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    expect(wrapper.text()).toContain('功能建设中')
    expect(wrapper.text()).toContain('风控事件')
    expect(wrapper.text()).toContain('风险规则')
    expect(wrapper.text()).toContain('人工复核')
    expect(wrapper.text()).toContain('当前页面不会执行任何管理操作。')
  })
})
