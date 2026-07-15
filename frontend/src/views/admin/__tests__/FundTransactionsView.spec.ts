import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import FundTransactionsView from '../FundTransactionsView.vue'

const messages: Record<string, string> = {
  'longdao.adminConstruction.badge': '功能建设中',
  'longdao.adminConstruction.futureCapabilities': '未来能力',
  'longdao.adminConstruction.statusTitle': '接入状态',
  'longdao.adminConstruction.statusDescription': '后台服务尚未接入。',
  'longdao.adminConstruction.disabledNotice': '当前页面不会执行任何管理操作。',
  'longdao.adminConstruction.fundTransactions.title': '资金流水',
  'longdao.adminConstruction.fundTransactions.description': '用于审计余额和资金变动。',
  'longdao.adminConstruction.fundTransactions.capabilities.ledger': '流水台账',
  'longdao.adminConstruction.fundTransactions.capabilities.adjustments': '余额调整',
  'longdao.adminConstruction.fundTransactions.capabilities.auditTrail': '审计追踪',
  'longdao.adminConstruction.fundTransactions.capabilities.export': '流水导出',
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

describe('FundTransactionsView', () => {
  it('renders the shared construction state and future fund transaction capabilities', () => {
    const wrapper = mount(FundTransactionsView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    expect(wrapper.text()).toContain('功能建设中')
    expect(wrapper.text()).toContain('资金流水')
    expect(wrapper.text()).toContain('流水台账')
    expect(wrapper.text()).toContain('审计追踪')
    expect(wrapper.text()).toContain('当前页面不会执行任何管理操作。')
  })
})
