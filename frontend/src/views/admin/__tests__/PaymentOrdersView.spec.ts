import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import PaymentOrdersView from '../PaymentOrdersView.vue'

const messages: Record<string, string> = {
  'longdao.adminConstruction.badge': '功能建设中',
  'longdao.adminConstruction.futureCapabilities': '未来能力',
  'longdao.adminConstruction.statusTitle': '接入状态',
  'longdao.adminConstruction.statusDescription': '后台服务尚未接入。',
  'longdao.adminConstruction.disabledNotice': '当前页面不会执行任何管理操作。',
  'longdao.adminConstruction.paymentOrders.title': '支付订单',
  'longdao.adminConstruction.paymentOrders.description': '用于查看和处理支付订单。',
  'longdao.adminConstruction.paymentOrders.capabilities.search': '订单检索',
  'longdao.adminConstruction.paymentOrders.capabilities.reconciliation': '支付对账',
  'longdao.adminConstruction.paymentOrders.capabilities.refunds': '退款处理',
  'longdao.adminConstruction.paymentOrders.capabilities.export': '订单导出',
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

describe('PaymentOrdersView', () => {
  it('renders the shared construction state and future payment order capabilities', () => {
    const wrapper = mount(PaymentOrdersView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    expect(wrapper.text()).toContain('功能建设中')
    expect(wrapper.text()).toContain('支付订单')
    expect(wrapper.text()).toContain('订单检索')
    expect(wrapper.text()).toContain('支付对账')
    expect(wrapper.text()).toContain('当前页面不会执行任何管理操作。')
  })
})
