import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import RechargeView from '../RechargeView.vue'

const messages: Record<string, string> = {
  'longdao.finance.recharge.badge': '财务中心',
  'longdao.finance.recharge.title': '账户充值',
  'longdao.finance.recharge.description': '选择金额和支付方式，等待支付后端接入。',
  'longdao.finance.recharge.currentBalance': '当前余额',
  'longdao.finance.recharge.balanceHint': '余额仅展示真实账户数据。',
  'longdao.finance.recharge.accountStatus': '接入状态',
  'longdao.finance.recharge.notConnected': '未接入',
  'longdao.finance.recharge.currency': '结算币种',
  'longdao.finance.recharge.amountTitle': '充值金额',
  'longdao.finance.recharge.customAmount': '自定义金额',
  'longdao.finance.recharge.customAmountPlaceholder': '请输入金额',
  'longdao.finance.recharge.paymentMethodTitle': '支付方式',
  'longdao.finance.recharge.paymentMethods.card': '银行卡',
  'longdao.finance.recharge.paymentMethods.wallet': '数字钱包',
  'longdao.finance.recharge.paymentMethodPending': '支付通道建设中',
  'longdao.finance.recharge.integrationNotice': '支付后端尚未接入，当前不会创建订单、扣款或变更余额。',
  'longdao.finance.recharge.disabledButton': '支付未接入，暂不可充值',
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

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: { balance: 12.34 },
  }),
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: { template: '<main><slot /></main>' },
}))

const AppLayoutStub = { template: '<main><slot /></main>' }

describe('RechargeView', () => {
  it('shows balance, amount choices, payment placeholders, and a disabled payment button', async () => {
    const wrapper = mount(RechargeView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
        },
      },
    })

    expect(wrapper.text()).toContain('账户充值')
    expect(wrapper.text()).toContain('¥12.34')
    expect(wrapper.text()).toContain('¥100')
    expect(wrapper.text()).toContain('¥1000')
    expect(wrapper.text()).toContain('银行卡')
    expect(wrapper.text()).toContain('支付后端尚未接入，当前不会创建订单、扣款或变更余额。')

    const customAmount = wrapper.find('input[type="number"]')
    await customAmount.setValue('88')
    expect((customAmount.element as HTMLInputElement).value).toBe('88')

    const disabledButton = wrapper.find('button[disabled][aria-disabled="true"]')
    expect(disabledButton.exists()).toBe(true)
    expect(disabledButton.text()).toBe('支付未接入，暂不可充值')
  })
})
