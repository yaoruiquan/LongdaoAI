// 龙道 AI 首页专属文案 (zh)
export default {
  brandSubtitle: '企业级 AI API',
  register: '注册',
  heroEyebrow: '龙道集团 · 企业级 AI API 服务平台',
  heroTitle: '稳定连接全球 AI 能力',
  primaryCta: '创建账号',
  secondaryCta: '查看接入方式',
  nav: {
    capabilities: '产品能力',
    models: '支持模型',
    billing: '计费优势'
  },
  metrics: {
    compatibleValue: 'OpenAI',
    compatibleLabel: '兼容接口',
    billingValue: '¥ / $',
    billingLabel: '人民币结算，美元参考',
    securityValue: '审计',
    securityLabel: '密钥与用量可追踪'
  },
  capabilities: {
    kicker: '平台能力',
    title: '统一接入、稳定路由、透明计量、安全控制',
    unified: {
      title: '统一接入',
      description: '以兼容 API 方式接入多类 AI 能力，减少多平台账号与密钥管理成本。'
    },
    routing: {
      title: '稳定路由',
      description: '围绕可用性、额度和策略进行请求调度，为业务接入保留稳定冗余。'
    },
    metering: {
      title: '透明计量',
      description: '用量、费用与额度关系清晰展示，支持按人民币结算并保留美元源成本参考。'
    },
    security: {
      title: '安全控制',
      description: '通过 API Key、权限隔离和日志审计，帮助团队安全地使用 AI 能力。'
    }
  },
  models: {
    kicker: '支持模型',
    title: '面向兼容能力展示模型类别',
    description: '首页仅展示平台计划支持的模型类别和兼容能力，不承诺未经验证的供应商可用性。',
    openai: {
      title: 'OpenAI 兼容模型',
      description: '适配 Chat Completions 等常见调用模式。'
    },
    claude: {
      title: 'Claude 类模型',
      description: '支持企业常用文本、代码和推理场景。'
    },
    gemini: {
      title: 'Gemini 类模型',
      description: '为多模态与长上下文场景保留接入扩展。'
    },
    compatible: {
      title: '更多兼容上游',
      description: '通过统一密钥与统一用量视图管理后续渠道。'
    }
  },
  integration: {
    kicker: '集成步骤',
    title: '三步完成 API 接入',
    createKey: {
      title: '创建 API Key',
      description: '登录控制台创建密钥，并按团队、项目或用途管理权限与额度。'
    },
    replaceBaseUrl: {
      title: '替换 Base URL',
      description: '在现有 OpenAI 或兼容 SDK 中替换基础地址，不在前端硬编码生产域名。'
    },
    sendRequest: {
      title: '发起请求并查看用量',
      description: '使用 Bearer 鉴权调用接口，在控制台查看消耗、成本和额度状态。'
    }
  },
  billing: {
    kicker: '计费优势',
    title: '清晰、诚实、可审计的费用表达',
    description: '本阶段仅展示目标规则说明，不伪造支付成功、余额变化或订单数据。',
    rule: '目标规则：套餐额度优先消耗，额度用尽后使用余额。后端计费规则将在后续阶段实现。',
    rmb: {
      title: '人民币结算',
      description: '面向用户以人民币金额为主展示，美元成本作为源成本或参考。'
    },
    transparent: {
      title: '透明明细',
      description: '用量与消费页面保留请求、Token、模型和成本维度。'
    },
    packageFirst: {
      title: '套餐优先',
      description: '清楚说明套餐额度与余额之间的消耗顺序，避免误导。'
    },
    support: {
      title: '服务保障',
      description: '围绕稳定性、权限隔离、日志审计和技术支持进行企业级表达。'
    }
  },
  finalCta: {
    title: '准备把 AI 能力接入业务？',
    description: '进入龙道 AI 控制台创建密钥、查看套餐和用量，按阶段完成企业级接入。',
    button: '立即注册'
  },
  viewOnGithub: '在 GitHub 上查看',
  viewDocs: '查看文档',
  docs: '文档',
  switchToLight: '切换到浅色模式',
  switchToDark: '切换到深色模式',
  dashboard: '控制台',
  login: '登录',
  getStarted: '立即开始',
  goToDashboard: '进入控制台',
  // 新增：面向用户的价值主张
  heroSubtitle: '一个密钥，畅用多个 AI 模型',
  heroDescription: '龙道集团为开发者与企业提供安全、稳定、透明计费的 AI API 接入服务。',
  tags: {
    subscriptionToApi: '订阅转 API',
    stickySession: '会话保持',
    realtimeBilling: '按量计费'
  },
  // 用户痛点区块
  painPoints: {
    title: '你是否也遇到这些问题？',
    items: {
      expensive: {
        title: '订阅费用高',
        desc: '每个 AI 服务都要单独订阅，每月支出越来越多'
      },
      complex: {
        title: '多账号难管理',
        desc: '不同平台的账号、密钥分散各处，管理起来很麻烦'
      },
      unstable: {
        title: '服务不稳定',
        desc: '单一账号容易触发限制，影响正常使用'
      },
      noControl: {
        title: '用量无法控制',
        desc: '不知道钱花在哪了，也无法限制团队成员的使用'
      }
    }
  },
  // 解决方案区块
  solutions: {
    title: '我们帮你解决',
    subtitle: '简单三步，开始省心使用 AI'
  },
  features: {
    unifiedGateway: '一键接入',
    unifiedGatewayDesc: '获取一个 API 密钥，即可调用所有已接入的 AI 模型，无需分别申请。',
    multiAccount: '稳定可靠',
    multiAccountDesc: '智能调度多个上游账号，自动切换和负载均衡，告别频繁报错。',
    balanceQuota: '用多少付多少',
    balanceQuotaDesc: '按实际使用量计费，支持设置配额上限，团队用量一目了然。'
  },
  // 优势对比
  comparison: {
    title: '为什么选择我们？',
    headers: {
      feature: '对比项',
      official: '官方订阅',
      us: '本平台'
    },
    items: {
      pricing: {
        feature: '付费方式',
        official: '固定月费，用不完也付',
        us: '按量付费，用多少付多少'
      },
      models: {
        feature: '模型选择',
        official: '单一服务商',
        us: '多模型随意切换'
      },
      management: {
        feature: '账号管理',
        official: '每个服务单独管理',
        us: '统一密钥，一站管理'
      },
      stability: {
        feature: '服务稳定性',
        official: '单账号易触发限制',
        us: '多账号池，自动切换'
      },
      control: {
        feature: '用量控制',
        official: '无法限制',
        us: '可设配额、查明细'
      }
    }
  },
  providers: {
    title: '已支持的 AI 模型',
    description: '一个 API，多种选择',
    supported: '已支持',
    soon: '即将推出',
    claude: 'Claude',
    gemini: 'Gemini',
    antigravity: 'Antigravity',
    more: '更多'
  },
  // CTA 区块
  cta: {
    title: '准备好开始了吗？',
    description: '注册即可获得免费试用额度，体验一站式 AI 服务',
    button: '免费注册'
  },
  footer: {
    allRightsReserved: '保留所有权利。'
  }
}
