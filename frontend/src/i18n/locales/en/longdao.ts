// 龙道 AI specific translations - English
export default {
  ai: 'AI',
  nav: {
    workbench: 'Workbench',
    apiServices: 'API Services',
    usageAndBilling: 'Usage & Spending',
    fundsAndPlans: 'Funds & Plans',
    planCenter: 'Plan Center',
    balanceRecharge: 'Balance Recharge',
    orderRecords: 'Order Records',
    redeemCode: 'Redeem Code',
    accountSecurity: 'Account Security',
    accountSettings: 'Account Settings',
    moreServices: 'More Services',
    userWorkbench: 'User Workbench',
    operationsOverview: 'Operations Overview',
    adminOverview: 'Admin Overview',
    usersAndBenefits: 'Users & Entitlements',
    resourcesAndChannels: 'Resources & Channels',
    channelAccounts: 'Channel Accounts',
    finance: 'Finance',
    usageLedger: 'Usage Ledger',
    paymentOrders: 'Payment Orders',
    fundTransactions: 'Fund Transactions',
    riskAndSystem: 'Risk & System',
    riskEvents: 'Risk Events',
    dataManagement: 'Data Management',
    dataManagementDescription: 'Manage system data, migrations, and maintenance tasks',
    backup: 'Backup & Restore',
    backupDescription: 'Manage database backup and restore operations'
  },
  dashboard: {
    badge: 'Enterprise Workbench',
    welcome: 'Welcome back',
    developer: 'Developer',
    description: 'Manage API keys, plan quota, balance, and request usage in one place.',
    createKey: 'Create API Key',
    viewPlans: 'View Plans',
    billingRuleTitle: 'Target Billing Priority',
    billingRuleDescription: 'Plan quota is consumed first, then account balance. This is a target rule only; backend billing logic will be implemented in a later phase.',
    loadFailed: 'Failed to load workbench data. Please retry.',
    planCenter: 'Plan Center',
    planCenterDesc: 'Review available plans and subscriptions',
    balanceRecharge: 'Balance Recharge',
    balanceRechargeDesc: 'Review balance and payment integration status'
  },
  apiGuide: {
    badge: 'Quick Start',
    title: 'Use a Longdao AI API Key',
    description: 'Connect through the unified Base URL with a compatible SDK or cURL and Bearer authentication.',
    copy: 'Copy',
    authHint: 'Use Authorization: Bearer YOUR_LONGDAO_API_KEY in request headers.',
    tabsLabel: 'API integration examples',
    copyCode: 'Copy code',
    serverOnly: 'Server-side only',
    serverOnlyDesc: 'Never expose API keys in browsers, mobile bundles, or public repositories.',
    rotateKeys: 'Rotate keys',
    rotateKeysDesc: 'Separate keys by project and disable any key immediately if it may be exposed.',
    leastPrivilege: 'Least privilege',
    leastPrivilegeDesc: 'Use separate limits, expiration dates, and access scopes for each workload.',
    copied: 'Copied to clipboard'
  },
  plans: {
    badge: 'Funds & Plans',
    title: 'Longdao AI Plan Center',
    description: 'Review plan capabilities, active subscriptions, and target billing rules. Real checkout opens only after payment integration.',
    balance: 'Current Balance',
    goRecharge: 'Go to Recharge',
    billingPriority: 'Plan quota is consumed first; account balance is used after quota is exhausted.',
    paymentPendingTitle: 'Payment integration in progress',
    paymentPendingDescription: 'No payment order, charge, or balance change will be created. Plan buttons only preview the product structure.',
    availablePlans: 'Available Plans',
    availablePlansDesc: 'These are frontend plan previews. Pricing, quota, and entitlements will be configured by operations.',
    frontendPreview: 'Plan Preview',
    recommended: 'Recommended',
    price: 'Plan Price',
    configuredByOperations: 'Configured by operations',
    paymentConnecting: 'Payment integration pending',
    currentSubscriptions: 'Current Subscriptions',
    currentSubscriptionsDesc: 'Displays real subscription data already associated with this account.',
    viewDetails: 'View subscription details',
    unnamedPlan: 'Unnamed plan',
    subscriptionActive: 'Subscription active',
    monthlyQuota: 'Monthly quota (USD source quota)',
    multiplier: 'Billing multiplier',
    noSubscription: 'No active subscription',
    noSubscriptionDesc: 'Real plan purchases will become available after payment backend integration.',
    unlimitedOrConfigured: 'Unlimited or operations-configured',
    viewPlanCenter: 'View Plan Center',
    starter: {
      title: 'Starter',
      description: 'For individual developers and small validation projects.',
      feature1: 'Unified API key management',
      feature2: 'Basic usage and spending records',
      feature3: 'Standard integration support'
    },
    professional: {
      title: 'Professional',
      description: 'For production products and development teams.',
      feature1: 'Higher quota and concurrency configuration',
      feature2: 'Team-oriented key and usage management',
      feature3: 'Priority technical support'
    },
    enterprise: {
      title: 'Enterprise',
      description: 'For organizations requiring governance, auditability, and dedicated service.',
      feature1: 'Enterprise access control and auditing',
      feature2: 'Dedicated quota and channel strategies',
      feature3: 'Business and technical service support'
    }
  },
  finance: {
    recharge: {
      badge: 'Finance Center',
      title: 'Balance Recharge',
      description: 'Review the RMB ledger balance and payment integration status. Real recharge opens after backend integration.',
      currentBalance: 'Current Ledger Balance',
      balanceHint: 'Only existing account data is displayed; this page cannot change the balance.',
      accountStatus: 'Payment Status',
      notConnected: 'Payment integration in progress',
      currency: 'Ledger Currency',
      amountTitle: 'Select Recharge Amount',
      customAmount: 'Custom Amount',
      customAmountPlaceholder: 'Enter an RMB amount',
      paymentMethodTitle: 'Payment Method',
      paymentMethodPending: 'Channel not connected',
      integrationNotice: 'Payment integration is in progress. No order, charge, or balance change will be created.',
      disabledButton: 'Recharge unavailable until payment integration',
      paymentMethods: {
        card: 'Bank Card',
        wallet: 'Enterprise Digital Wallet'
      }
    },
    orders: {
      badge: 'Finance Center',
      title: 'Order Records',
      description: 'Real order status and payment records will appear here after payment backend integration.',
      emptyTitle: 'No real orders',
      emptyDescription: 'The payment system is not connected, so there are no real orders to display and no mock orders are generated.'
    }
  },
  adminConstruction: {
    badge: 'Under Construction',
    futureCapabilities: 'Planned Capabilities',
    statusTitle: 'Current Status',
    statusDescription: 'The page structure is ready. Backend APIs, permissions, and data models will be connected in the relevant phase.',
    disabledNotice: 'This is a read-only construction state with no actions that create business data.',
    paymentOrders: {
      title: 'Payment Order Management',
      description: 'Will support real payment-order search, reconciliation, refunds, and exports.',
      capabilities: {
        search: 'Order search and status filters',
        reconciliation: 'Payment-channel reconciliation',
        refunds: 'Refund and exception handling',
        export: 'Finance record exports'
      }
    },
    fundTransactions: {
      title: 'Fund Transactions',
      description: 'Will record balance credits, deductions, adjustments, and audit history.',
      capabilities: {
        ledger: 'User fund ledger',
        adjustments: 'Manual adjustments and approvals',
        auditTrail: 'Tamper-evident audit trail',
        export: 'Transaction export and reconciliation'
      }
    },
    riskEvents: {
      title: 'Risk Events',
      description: 'Will aggregate anomalous requests, account behavior, and risk-control actions.',
      capabilities: {
        rules: 'Risk rules and trigger reasons',
        alerts: 'Real-time alerts and notifications',
        reviewQueue: 'Manual review queue',
        reporting: 'Risk trends and reports'
      }
    }
  }
}
