import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoginView from '../LoginView.vue'

const { getPublicSettings, login, login2FA, push } = vi.hoisted(() => ({
  getPublicSettings: vi.fn(),
  login: vi.fn(),
  login2FA: vi.fn(),
  push: vi.fn()
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings,
  isTotp2FARequired: (response: { requires_2fa?: boolean }) => response?.requires_2fa === true
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({ login, login2FA }),
  useAppStore: () => ({ showSuccess: vi.fn(), showError: vi.fn() })
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push, currentRoute: { value: { query: {} } } })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'auth.loginEyebrow': '安全登录',
    'auth.welcomeBack': '欢迎回来',
    'auth.signInToAccount': '登录您的账户以继续',
    'auth.emailLabel': '邮箱',
    'auth.emailPlaceholder': '请输入邮箱',
    'auth.passwordLabel': '密码',
    'auth.passwordPlaceholder': '请输入密码',
    'auth.forgotPassword': '忘记密码',
    'auth.signIn': '登录',
    'auth.signingIn': '登录中',
    'auth.securityNote': '安全提示',
    'auth.dontHaveAccount': '还没有账户？',
    'auth.createAccount': '创建账户'
  }
  return { ...actual, useI18n: () => ({ t: (key: string) => messages[key] ?? key }) }
})

const AuthLayoutStub = {
  template: '<main><slot /><footer><slot name="footer" /></footer></main>'
}

const TotpLoginModalStub = {
  template: '<div data-test="totp-modal">2FA</div>'
}

describe('LoginView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getPublicSettings.mockResolvedValue({
      password_reset_enabled: true,
      turnstile_enabled: false,
      turnstile_site_key: '',
      linuxdo_oauth_enabled: false,
      backend_mode_enabled: false
    })
  })

  it('does not show the 2FA modal before the backend requests it', async () => {
    const wrapper = mount(LoginView, {
      global: {
        stubs: {
          AuthLayout: AuthLayoutStub,
          LinuxDoOAuthSection: true,
          TurnstileWidget: true,
          TotpLoginModal: TotpLoginModalStub,
          Icon: true,
          RouterLink: { template: '<a><slot /></a>' }
        }
      }
    })

    await flushPromises()

    expect(wrapper.find('[data-test="totp-modal"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('还没有账户？')
    expect(wrapper.text()).not.toContain('auth.noAccount')
  })
})
