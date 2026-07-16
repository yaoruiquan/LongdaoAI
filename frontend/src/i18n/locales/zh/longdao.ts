// 龙道 AI 专属文案 - 中文
export default {
  ai: 'AI',
  nav: {
    workbench: '工作台',
    apiServices: 'API 服务',
    usageAndBilling: '用量与消费',
    fundsAndPlans: '资金与套餐',
    planCenter: '套餐中心',
    balanceRecharge: '余额充值',
    orderRecords: '订单记录',
    redeemCode: '兑换码',
    accountSecurity: '账户安全',
    accountSettings: '账户设置',
    moreServices: '更多服务',
    userWorkbench: '用户工作台',
    operationsOverview: '运营总览',
    adminOverview: '管理概览',
    usersAndBenefits: '用户与权益',
    resourcesAndChannels: '资源与渠道',
    channelAccounts: '渠道账户',
    finance: '财务管理',
    usageLedger: '用量账单',
    paymentOrders: '支付订单',
    fundTransactions: '资金流水',
    riskAndSystem: '风控与系统',
    riskEvents: '风险事件',
    dataManagement: '数据管理',
    dataManagementDescription: '管理系统数据、迁移与维护任务',
    backup: '备份与恢复',
    backupDescription: '管理数据库备份与恢复操作'
  },
  dashboard: {
    badge: '企业工作台',
    welcome: '欢迎回来',
    developer: '开发者',
    description: '统一管理 API Key、套餐额度、余额与调用用量。',
    createKey: '创建 API Key',
    viewPlans: '查看套餐',
    billingRuleTitle: '目标计费顺序',
    billingRuleDescription: '套餐额度优先消耗，额度用尽后使用账户余额。当前页面仅展示目标规则，后端计费逻辑将在后续阶段接入。',
    loadFailed: '工作台数据加载失败，请重试。',
    planCenter: '套餐中心',
    planCenterDesc: '查看可用套餐和当前订阅',
    balanceRecharge: '余额充值',
    balanceRechargeDesc: '查看余额和支付接入状态'
  },
  apiGuide: {
    badge: '快速接入',
    title: '使用龙道 AI API Key',
    description: '使用兼容 SDK 或 cURL，通过统一 Base URL 和 Bearer 鉴权接入模型能力。',
    copy: '复制',
    authHint: '请求头使用 Authorization: Bearer YOUR_LONGDAO_API_KEY。',
    tabsLabel: 'API 接入示例',
    copyCode: '复制代码',
    serverOnly: '仅服务端使用',
    serverOnlyDesc: '不要把 API Key 写入浏览器、移动端安装包或公开仓库。',
    rotateKeys: '定期轮换密钥',
    rotateKeysDesc: '按项目拆分密钥，发现泄露后立即禁用并重新创建。',
    leastPrivilege: '最小权限',
    leastPrivilegeDesc: '为不同业务设置独立额度、有效期和访问范围。',
    copied: '已复制到剪贴板'
  },
  plans: {
    badge: '资金与套餐',
    title: '龙道 AI 套餐中心',
    description: '查看套餐能力、当前订阅与目标计费规则。真实支付和下单将在支付系统接入后开放。',
    balance: '当前余额',
    goRecharge: '前往充值',
    billingPriority: '套餐额度优先消耗，额度用尽后使用余额。',
    paymentPendingTitle: '支付系统接入中',
    paymentPendingDescription: '当前不会创建支付订单、扣款或变更余额；套餐按钮仅用于展示产品结构。',
    availablePlans: '可用套餐',
    availablePlansDesc: '以下为前端展示方案，具体价格、额度和权益由运营后台配置。',
    frontendPreview: '方案预览',
    recommended: '推荐',
    price: '套餐价格',
    configuredByOperations: '由运营配置',
    paymentConnecting: '支付接入中',
    currentSubscriptions: '当前订阅',
    currentSubscriptionsDesc: '读取账户现有的真实订阅数据。',
    viewDetails: '查看订阅详情',
    unnamedPlan: '未命名套餐',
    subscriptionActive: '订阅已生效',
    monthlyQuota: '月度额度（USD 源额度）',
    multiplier: '计费倍率',
    noSubscription: '暂无有效订阅',
    noSubscriptionDesc: '支付后端接入后，可在套餐中心完成真实购买。',
    unlimitedOrConfigured: '不限或由运营配置',
    viewPlanCenter: '查看套餐中心',
    starter: {
      title: '入门版',
      description: '适合个人开发者与小型验证项目。',
      feature1: '统一 API Key 管理',
      feature2: '基础用量与消费记录',
      feature3: '标准接入支持'
    },
    professional: {
      title: '专业版',
      description: '适合稳定运行的产品与开发团队。',
      feature1: '更高额度与并发配置',
      feature2: '团队化密钥和用量管理',
      feature3: '优先技术支持'
    },
    enterprise: {
      title: '企业版',
      description: '适合需要权限、审计和专属保障的企业。',
      feature1: '企业级权限与审计能力',
      feature2: '专属额度和渠道策略',
      feature3: '商务与技术服务支持'
    }
  },
  finance: {
    recharge: {
      badge: '财务中心',
      title: '余额充值',
      description: '查看人民币账面余额与支付系统接入状态。真实充值将在支付后端完成后开放。',
      currentBalance: '当前账面余额',
      balanceHint: '余额仅展示账户现有数据，本页面不会修改余额。',
      accountStatus: '支付状态',
      notConnected: '支付系统接入中',
      currency: '账面币种',
      amountTitle: '选择充值金额',
      customAmount: '自定义金额',
      customAmountPlaceholder: '请输入人民币金额',
      paymentMethodTitle: '支付方式',
      paymentMethodPending: '通道尚未接入',
      integrationNotice: '支付系统接入中。当前不会创建订单、发起扣款或变更账户余额。',
      disabledButton: '支付未接入，暂不可充值',
      paymentMethods: {
        card: '银行卡支付',
        wallet: '企业数字钱包'
      }
    },
    orders: {
      badge: '财务中心',
      title: '订单记录',
      description: '支付后端接入后，可在此查看真实订单状态和支付记录。',
      emptyTitle: '暂无真实订单',
      emptyDescription: '支付系统尚未接入，因此当前没有可展示的真实订单，也不会生成模拟订单。'
    }
  },
  adminConstruction: {
    badge: '功能建设中',
    futureCapabilities: '计划能力',
    statusTitle: '当前状态',
    statusDescription: '页面结构已完成，后端接口、权限和数据模型将在对应阶段接入。',
    disabledNotice: '当前为只读建设中状态，不提供会产生业务数据的操作。',
    paymentOrders: {
      title: '支付订单管理',
      description: '后续用于查询真实支付订单、对账、退款和导出。',
      capabilities: {
        search: '订单查询与状态筛选',
        reconciliation: '支付通道对账',
        refunds: '退款与异常处理',
        export: '财务记录导出'
      }
    },
    fundTransactions: {
      title: '资金流水',
      description: '后续用于记录余额入账、扣减、调整和审计轨迹。',
      capabilities: {
        ledger: '用户资金台账',
        adjustments: '人工调整与审批',
        auditTrail: '不可抵赖审计记录',
        export: '流水导出与核对'
      }
    },
    riskEvents: {
      title: '风险事件',
      description: '后续用于汇总异常请求、账户行为和风控处置记录。',
      capabilities: {
        rules: '风险规则与命中原因',
        alerts: '实时告警和通知',
        reviewQueue: '人工复核队列',
        reporting: '风险趋势与报表'
      }
    }
  }
}
