import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import OrdersView from '../OrdersView.vue'

const messages: Record<string, string> = {
  'common.noData': '暂无数据',
  'longdao.finance.orders.badge': '财务中心',
  'longdao.finance.orders.title': '订单记录',
  'longdao.finance.orders.description': '查看真实支付订单。',
  'longdao.finance.orders.emptyTitle': '暂无订单',
  'longdao.finance.orders.emptyDescription': '支付后端尚未接入，因此当前没有可展示的真实订单。',
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

describe('OrdersView', () => {
  it('renders a real empty state without fake orders', () => {
    const wrapper = mount(OrdersView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          Icon: true,
        },
      },
    })

    expect(wrapper.text()).toContain('订单记录')
    expect(wrapper.text()).toContain('暂无订单')
    expect(wrapper.text()).toContain('支付后端尚未接入，因此当前没有可展示的真实订单。')
    expect(wrapper.find('tbody tr').exists()).toBe(false)
  })
})
