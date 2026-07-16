import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import PurchaseSubscriptionView from '../PurchaseSubscriptionView.vue'

const { getMySubscriptions, refreshUser, showError } = vi.hoisted(() => ({
  getMySubscriptions: vi.fn().mockResolvedValue([]),
  refreshUser: vi.fn().mockResolvedValue(undefined),
  showError: vi.fn()
}))

vi.mock('@/api/subscriptions', () => ({
  default: { getMySubscriptions }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: { balance: 88.5 },
    refreshUser
  })
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const AppLayoutStub = { template: '<main><slot /></main>' }
const RouterLinkStub = { template: '<a><slot /></a>' }

describe('PurchaseSubscriptionView', () => {
  it('shows existing account data while keeping every purchase action disabled', async () => {
    const wrapper = mount(PurchaseSubscriptionView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          RouterLink: RouterLinkStub,
          LoadingSpinner: true,
          RuleNotice: { props: ['title', 'description'], template: '<div>{{ title }} {{ description }}</div>' }
        }
      }
    })

    await flushPromises()

    expect(refreshUser).toHaveBeenCalledOnce()
    expect(getMySubscriptions).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('¥88.50')
    expect(wrapper.text()).toContain('longdao.plans.paymentPendingTitle')

    const purchaseButtons = wrapper.findAll('button')
    expect(purchaseButtons.length).toBeGreaterThan(0)
    expect(purchaseButtons.every((button) => button.attributes('disabled') !== undefined)).toBe(true)
    expect(showError).not.toHaveBeenCalled()
  })
})
