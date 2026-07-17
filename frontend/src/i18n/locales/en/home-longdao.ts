// 龙道 AI 首页专属文案 (en)
export default {
  brandSubtitle: 'Enterprise AI API',
  register: 'Register',
  heroEyebrow: 'Longdao Group · Enterprise AI API Platform',
  heroTitle: 'Stable access to global AI capabilities',
  primaryCta: 'Create account',
  secondaryCta: 'View integration',
  nav: {
    capabilities: 'Capabilities',
    models: 'Models',
    billing: 'Billing'
  },
  metrics: {
    compatibleValue: 'OpenAI',
    compatibleLabel: 'Compatible API',
    billingValue: '¥ / $',
    billingLabel: 'RMB billing, USD reference',
    securityValue: 'Audit',
    securityLabel: 'Keys and usage traceable'
  },
  capabilities: {
    kicker: 'Platform Capabilities',
    title: 'Unified access, stable routing, transparent metering, secure control',
    unified: {
      title: 'Unified Access',
      description: 'Access multiple AI capability categories through compatible APIs and reduce account and key management overhead.'
    },
    routing: {
      title: 'Stable Routing',
      description: 'Route requests around availability, quota, and policy to keep redundancy for business integrations.'
    },
    metering: {
      title: 'Transparent Metering',
      description: 'Present usage, cost, and quota relationships clearly, with RMB billing and USD source-cost reference.'
    },
    security: {
      title: 'Secure Control',
      description: 'Use API keys, permission isolation, and audit logs to help teams consume AI capabilities safely.'
    }
  },
  models: {
    kicker: 'Supported Models',
    title: 'Model categories shown by compatibility',
    description: 'The homepage describes planned model categories and compatibility only, without promising unverified provider availability.',
    openai: {
      title: 'OpenAI-compatible models',
      description: 'Adapt common call patterns such as Chat Completions.'
    },
    claude: {
      title: 'Claude-style models',
      description: 'Support common enterprise text, code, and reasoning scenarios.'
    },
    gemini: {
      title: 'Gemini-style models',
      description: 'Reserve expansion paths for multimodal and long-context use cases.'
    },
    compatible: {
      title: 'More compatible upstreams',
      description: 'Manage future channels with one key and one usage view.'
    }
  },
  integration: {
    kicker: 'Integration Steps',
    title: 'Connect APIs in three steps',
    createKey: {
      title: 'Create an API Key',
      description: 'Sign in to the console, create keys, and manage permissions and quotas by team, project, or purpose.'
    },
    replaceBaseUrl: {
      title: 'Replace Base URL',
      description: 'Update the base URL in existing OpenAI or compatible SDKs without hardcoding production domains in the frontend.'
    },
    sendRequest: {
      title: 'Send requests and review usage',
      description: 'Call APIs with Bearer auth and review spending, cost, and quota status in the console.'
    }
  },
  billing: {
    kicker: 'Billing Advantages',
    title: 'Clear, honest, auditable cost presentation',
    description: 'This phase only explains target rules and does not fake payment success, balance changes, or orders.',
    rule: 'Target rule: subscription quota is consumed first; balance is used after quota is exhausted. Backend billing behavior ships in a later phase.',
    rmb: {
      title: 'RMB settlement',
      description: 'User-facing amounts prioritize RMB, while USD remains source-cost or reference cost.'
    },
    transparent: {
      title: 'Transparent details',
      description: 'Usage and spending pages keep request, token, model, and cost dimensions visible.'
    },
    packageFirst: {
      title: 'Package first',
      description: 'Clearly explain the consumption order between package quota and balance.'
    },
    support: {
      title: 'Service assurance',
      description: 'Express enterprise readiness through stability, permission isolation, audit logs, and support.'
    }
  },
  finalCta: {
    title: 'Ready to connect AI to your business?',
    description: 'Enter the Longdao AI console to create keys, view packages, and track usage for phased enterprise integration.',
    button: 'Register now'
  },
  viewOnGithub: 'View on GitHub',
  viewDocs: 'View Documentation',
  docs: 'Docs',
  switchToLight: 'Switch to Light Mode',
  switchToDark: 'Switch to Dark Mode',
  dashboard: 'Dashboard',
  login: 'Login',
  getStarted: 'Get Started',
  goToDashboard: 'Go to Dashboard',
  // User-focused value proposition
  heroSubtitle: 'One Key, All AI Models',
  heroDescription: 'Longdao Group provides developers and enterprises with secure, stable AI API access and transparent billing.',
  tags: {
    subscriptionToApi: 'Subscription to API',
    stickySession: 'Session Persistence',
    realtimeBilling: 'Pay As You Go'
  },
  // Pain points section
  painPoints: {
    title: 'Sound Familiar?',
    items: {
      expensive: {
        title: 'High Subscription Costs',
        desc: 'Paying for multiple AI subscriptions that add up every month'
      },
      complex: {
        title: 'Account Chaos',
        desc: 'Managing scattered accounts and API keys across different platforms'
      },
      unstable: {
        title: 'Service Interruptions',
        desc: 'Single accounts hitting rate limits and disrupting your workflow'
      },
      noControl: {
        title: 'No Usage Control',
        desc: "Can't track where your money goes or limit team member usage"
      }
    }
  },
  // Solutions section
  solutions: {
    title: 'We Solve These Problems',
    subtitle: 'Three simple steps to stress-free AI access'
  },
  features: {
    unifiedGateway: 'One-Click Access',
    unifiedGatewayDesc: 'Get a single API key to call all connected AI models. No separate applications needed.',
    multiAccount: 'Always Reliable',
    multiAccountDesc: 'Smart routing across multiple upstream accounts with automatic failover. Say goodbye to errors.',
    balanceQuota: 'Pay What You Use',
    balanceQuotaDesc: 'Usage-based billing with quota limits. Full visibility into team consumption.'
  },
  // Comparison section
  comparison: {
    title: 'Why Choose Us?',
    headers: {
      feature: 'Comparison',
      official: 'Official Subscriptions',
      us: 'Our Platform'
    },
    items: {
      pricing: {
        feature: 'Pricing',
        official: 'Fixed monthly fee, pay even if unused',
        us: 'Pay only for what you use'
      },
      models: {
        feature: 'Model Selection',
        official: 'Single provider only',
        us: 'Switch between models freely'
      },
      management: {
        feature: 'Account Management',
        official: 'Manage each service separately',
        us: 'Unified key, one dashboard'
      },
      stability: {
        feature: 'Stability',
        official: 'Single account rate limits',
        us: 'Multi-account pool, auto-failover'
      },
      control: {
        feature: 'Usage Control',
        official: 'Not available',
        us: 'Quotas & detailed analytics'
      }
    }
  },
  providers: {
    title: 'Supported AI Models',
    description: 'One API, Multiple Choices',
    supported: 'Supported',
    soon: 'Soon',
    claude: 'Claude',
    gemini: 'Gemini',
    antigravity: 'Antigravity',
    more: 'More'
  },
  // CTA section
  cta: {
    title: 'Ready to Get Started?',
    description: 'Sign up now and get free trial credits to experience seamless AI access',
    button: 'Sign Up Free'
  },
  footer: {
    allRightsReserved: 'All rights reserved.'
  }
}
