# 龙道 AI 第一阶段前端改版 Implementation Plan

> **For Codex:** 按任务顺序实施；每个任务先锁定现有行为，再做最小改动并执行针对性验证。

**Goal:** 在不改变现有 API 行为的前提下，将 Sub2API 前端改造成龙道 AI 企业级 AI API 服务平台，并补齐充值、订单和管理占位界面。

**Architecture:** 保留 Vue 3、Pinia、Vue Router、Tailwind 和 vue-i18n。品牌令牌集中管理，公共视觉组件复用；已有业务页面继续使用原 store/API，尚无后端的数据页面使用诚实的只读或建设中状态。

**Tech Stack:** Vue 3、TypeScript、Pinia、Vue Router、vue-i18n、Tailwind CSS、Vitest、Vite、Docker。

---

### Task 1: 固化品牌资源和全局设计令牌

**Files:**
- Create: `frontend/public/favicon.ico`
- Create: `frontend/public/longdao-logo.png`
- Create: `frontend/src/components/brand/BrandLockup.vue`
- Modify: `frontend/index.html`
- Modify: `frontend/tailwind.config.js`
- Modify: `frontend/src/style.css`
- Modify: `frontend/src/stores/app.ts`
- Test: `frontend/src/stores/__tests__/app.spec.ts`

**Steps:**
1. 复制用户提供的 ICO，并生成透明 PNG。
2. 将默认站点标题、名称、Logo 和副标题改为龙道 AI 品牌，同时保留后端公开设置覆盖能力。
3. 将 Tailwind `primary` 色板、光晕、渐变和网格改为以 `#EB3F00` 为核心的红色体系。
4. 添加可复用品牌锁定组件，支持收起、完整、浅色和暗色场景。
5. 更新默认值相关测试。
6. 运行 `pnpm --dir frontend test:run src/stores/__tests__/app.spec.ts`、lint 和 typecheck。

### Task 2: 重构首页

**Files:**
- Modify: `frontend/src/views/HomeView.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: `frontend/src/__tests__/integration/navigation.spec.ts`
- Create if useful: `frontend/src/views/__tests__/HomeView.spec.ts`

**Steps:**
1. 为首页品牌文案和 CTA 添加中英文翻译。
2. 实现品牌导航、Hero、能力、模型、接入流程、计费优势、服务保障和页尾 CTA。
3. 避免伪造实时调用量或 SLA 数据；使用能力说明或明确的产品指标文案。
4. 保留登录态跳转和公开设置兼容。
5. 添加首页关键文案、链接和响应式结构测试。
6. 运行针对性测试、lint 和 typecheck。

### Task 3: 重构认证布局与登录注册页

**Files:**
- Modify: `frontend/src/components/layout/AuthLayout.vue`
- Modify: `frontend/src/views/auth/LoginView.vue`
- Modify: `frontend/src/views/auth/RegisterView.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: `frontend/src/components/__tests__/LoginForm.spec.ts`
- Create if useful: `frontend/src/views/auth/__tests__/RegisterView.spec.ts`

**Steps:**
1. 将认证布局改为桌面双栏、移动单栏的企业科技布局。
2. 左侧展示统一接入、安全稳定、透明计费和技术支持；右侧承载原表单 slot。
3. 保留登录的 TOTP、OAuth、找回密码和错误处理。
4. 保留注册的邮箱验证、邀请码、推广码、Turnstile 和公开注册开关逻辑。
5. 更新视觉文案和版权为龙道集团。
6. 运行登录与注册针对性测试，确保请求参数和条件渲染不变。

### Task 4: 重组控制台与管理员导航

**Files:**
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/components/layout/AppLayout.vue`
- Modify: `frontend/src/components/layout/AppHeader.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: `frontend/src/__tests__/integration/navigation.spec.ts`
- Test: `frontend/src/router/__tests__/guards.spec.ts`
- Test: `frontend/src/router/__tests__/title.spec.ts`

**Steps:**
1. 扩展导航项类型以支持分组和建设中状态。
2. 按已确认的信息架构重组用户菜单，并兼容简单模式、自定义菜单和 Sora 开关。
3. 按运营、用户、资源、财务、风控分组管理员菜单。
4. 新增充值、订单、支付订单、资金流水、风险事件路由。
5. 保持权限守卫和管理员个人菜单行为。
6. 更新导航、标题和路由守卫测试。

### Task 5: 改造工作台与资金摘要

**Files:**
- Create: `frontend/src/components/finance/BalanceOverviewCard.vue`
- Create: `frontend/src/components/common/RuleNotice.vue`
- Modify: `frontend/src/views/user/DashboardView.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: `frontend/src/components/__tests__/Dashboard.spec.ts`

**Steps:**
1. 复用现有用户、用量和订阅数据，构建余额、套餐、今日消费和 Key 状态卡片。
2. 添加“套餐优先、余额兜底”的目标规则说明。
3. 对不存在的后端字段显示说明或 `--`，不生成虚假数据。
4. 增加创建 Key、查看用量、购买套餐、充值的快捷入口。
5. 覆盖加载、空状态和 API 失败场景。
6. 运行 Dashboard 测试、lint 和 typecheck。

### Task 6: 增强 API Key 页面与接入指南

**Files:**
- Create: `frontend/src/components/common/CodeExampleTabs.vue`
- Create: `frontend/src/components/keys/ApiIntegrationGuide.vue`
- Modify: `frontend/src/views/user/KeysView.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: `frontend/src/components/__tests__/ApiKeyCreate.spec.ts`
- Create if useful: `frontend/src/components/keys/__tests__/ApiIntegrationGuide.spec.ts`

**Steps:**
1. 保留现有 Key CRUD 逻辑和组件。
2. 从当前页面来源或已有配置推导 Base URL。
3. 提供 OpenAI SDK、Claude SDK 和 cURL 示例及复制操作。
4. 添加密钥安全提示。
5. 为示例切换、复制内容和 Base URL 生成添加测试。
6. 运行 Key 相关测试、lint 和 typecheck。

### Task 7: 改造套餐中心和购买页

**Files:**
- Modify: `frontend/src/views/user/SubscriptionsView.vue`
- Modify: `frontend/src/views/user/PurchaseSubscriptionView.vue`
- Modify: `frontend/src/stores/subscriptions.ts` only if display selectors are needed
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: `frontend/src/stores/__tests__/subscriptions.spec.ts`
- Create if useful: `frontend/src/views/user/__tests__/PurchaseSubscriptionView.spec.ts`

**Steps:**
1. 保留现有订阅产品和用户订阅请求。
2. 统一套餐卡展示价格、额度、周期、倍率、模型和状态。
3. 在所有用户购买入口展示“支付系统接入中”。
4. 阻止未接入支付按钮产生成功提示或余额/订阅变化。
5. 测试已有数据展示和支付未接入状态。
6. 运行订阅相关测试、lint 和 typecheck。

### Task 8: 新增充值和订单页面

**Files:**
- Create: `frontend/src/views/user/RechargeView.vue`
- Create: `frontend/src/views/user/OrdersView.vue`
- Create: `frontend/src/components/common/ConstructionState.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Create: `frontend/src/views/user/__tests__/RechargeView.spec.ts`
- Create: `frontend/src/views/user/__tests__/OrdersView.spec.ts`

**Steps:**
1. 充值页展示当前余额、预设金额、自定义金额和支付方式占位。
2. 禁用真实支付动作并说明支付系统尚未接入。
3. 订单页展示订单字段定义、筛选外观和真实空状态，不构造订单。
4. 使用统一建设中组件表达后续能力。
5. 测试页面不可产生支付成功、余额变化或虚假订单。
6. 运行新增测试、lint 和 typecheck。

### Task 9: 增强用量与消费展示

**Files:**
- Modify: `frontend/src/views/user/UsageView.vue`
- Modify as needed: `frontend/src/utils/format.ts`
- Modify as needed: `frontend/src/utils/usagePricing.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Test: `frontend/src/views/user/__tests__/UsageView.spec.ts`
- Test as needed: `frontend/src/utils/__tests__/usageServiceTier.spec.ts`

**Steps:**
1. 保留现有用量查询和筛选。
2. 将用户侧主要金额统一为人民币格式。
3. 保留美元源成本参考，并明确标签避免混淆。
4. 对缺少汇率或人民币金额的数据使用现有金额并标注来源，不自行杜撰换算率。
5. 更新金额展示和空状态测试。
6. 运行 Usage 相关测试、lint 和 typecheck。

### Task 10: 新增管理员财务与风控占位页

**Files:**
- Create: `frontend/src/views/admin/PaymentOrdersView.vue`
- Create: `frontend/src/views/admin/FundTransactionsView.vue`
- Create: `frontend/src/views/admin/RiskEventsView.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`
- Create: `frontend/src/views/admin/__tests__/ConstructionViews.spec.ts`

**Steps:**
1. 为三个页面设置管理员权限路由和中英文标题。
2. 使用统一建设中组件展示功能范围和后续依赖。
3. 不添加无后端支持的操作按钮。
4. 验证普通用户不可访问、管理员可访问。
5. 运行路由和页面测试。

### Task 11: 全量验证和本地部署检查

**Files:**
- Modify only if verification exposes defects.

**Steps:**
1. 运行 `PATH=/opt/homebrew/opt/node@24/bin:$PATH pnpm --dir frontend run lint:check`。
2. 运行 `PATH=/opt/homebrew/opt/node@24/bin:$PATH pnpm --dir frontend run typecheck`。
3. 运行 `PATH=/opt/homebrew/opt/node@24/bin:$PATH pnpm --dir frontend run test:run`。
4. 运行 `PATH=/opt/homebrew/opt/node@24/bin:$PATH pnpm --dir frontend run build`。
5. 运行 `docker compose -f deploy/docker-compose.dev.yml up -d --build`。
6. 检查 `http://127.0.0.1:3000`、`http://127.0.0.1:8080` 和 `/health`。
7. 检查桌面和移动尺寸、浅色和暗色、中文和英文、登录注册、用户导航及管理员导航。
8. 记录任何因后端尚未实现而保留的限制。
