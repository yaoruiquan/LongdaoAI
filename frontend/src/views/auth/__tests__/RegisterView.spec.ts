import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RegisterView from '../RegisterView.vue'

const { getPublicSettings, validatePromoCode, validateInvitationCode, push } = vi.hoisted(() => ({
  getPublicSettings: vi.fn(),
  validatePromoCode: vi.fn(),
  validateInvitationCode: vi.fn(),
  push: vi.fn()
}))

vi.mock('@/api/auth', () => ({ getPublicSettings, validatePromoCode, validateInvitationCode }))
vi.mock('@/stores', () => ({
  useAuthStore: () => ({ register: vi.fn() }),
  useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() })
}))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
  useRoute: () => ({ query: {} })
}))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string, params?: Record<string, string>) => {
        if (key === 'auth.signUpToStart') return `注册以开始使用 ${params?.siteName ?? ''}`
        if (key === 'auth.registrationDisabled') return '注册功能暂时关闭，请联系管理员。'
        return key
      }
    })
  }
})

const AuthLayoutStub = {
  template: '<main><slot /><footer><slot name="footer" /></footer></main>'
}

describe('RegisterView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getPublicSettings.mockResolvedValue({
      registration_enabled: true,
      email_verify_enabled: false,
      promo_code_enabled: false,
      invitation_code_enabled: false,
      turnstile_enabled: false,
      turnstile_site_key: '',
      site_name: '龙道 AI',
      linuxdo_oauth_enabled: true,
      registration_email_suffix_whitelist: []
    })
  })

  it('keeps OAuth registration and renders the configured site name', async () => {
    const wrapper = mount(RegisterView, {
      global: {
        stubs: {
          AuthLayout: AuthLayoutStub,
          LinuxDoOAuthSection: { template: '<div data-test="oauth-registration" />' },
          TurnstileWidget: true,
          Icon: true,
          RouterLink: { template: '<a><slot /></a>' }
        }
      }
    })

    await flushPromises()

    expect(wrapper.find('[data-test="oauth-registration"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('注册以开始使用 龙道 AI')
    expect(wrapper.find('form').exists()).toBe(true)
  })

  it('hides the form after settings confirm registration is disabled', async () => {
    getPublicSettings.mockResolvedValueOnce({
      registration_enabled: false,
      email_verify_enabled: false,
      promo_code_enabled: false,
      invitation_code_enabled: false,
      turnstile_enabled: false,
      turnstile_site_key: '',
      site_name: '龙道 AI',
      linuxdo_oauth_enabled: false,
      registration_email_suffix_whitelist: []
    })

    const wrapper = mount(RegisterView, {
      global: {
        stubs: {
          AuthLayout: AuthLayoutStub,
          LinuxDoOAuthSection: true,
          TurnstileWidget: true,
          Icon: true,
          RouterLink: { template: '<a><slot /></a>' }
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('注册功能暂时关闭，请联系管理员。')
    expect(wrapper.find('form').exists()).toBe(false)
  })
})
