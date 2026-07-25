# AI API 中转站架构原理

> 面向有一定开发基础、想理解 LLM 中转站如何设计和运行的开发者。

---

## 1. 为什么需要中转站

直接调用 OpenAI 官方 API 是最简单的方式，但有一些现实问题：

- **地区限制**：中国大陆 IP 无法访问 OpenAI，每个用户都要自己解决网络问题
- **账号管理**：公司有多个 OpenAI 账号，想统一分配给员工使用
- **多模型统一入口**：用户只需要一个 `base_url` 和一个 API Key，背后可以无缝切换 OpenAI、Claude、Gemini
- **统一计费和用量追踪**：知道每个用户花了多少钱，控制每个 Key 的额度上限
- **限速控制**：防止某个用户把额度用完，保护整体服务可用性

中转站就是在用户和真正的 AI 服务之间插入的一层代理，它对用户暴露一个**兼容 OpenAI 格式**的接口，背后再转发给真正的提供商。

### 直连 vs 走中转的权衡

| 维度 | 直连官方 API | 走第三方聚合中转 |
|------|-------------|-----------------|
| 延迟 | 最低 | 多一跳，稍高 |
| 稳定性 | 官方保证 | 取决于中转商 |
| 价格 | 官方定价 | 通常更便宜（批发） |
| 模型覆盖 | 单一厂商 | 多厂商聚合 |
| 合规风险 | 低 | 需评估中转商资质 |

本次部署选择的架构是：自建 sub2api 实例（开源项目）+ 上游聚合中转站（aipro.hk.cn），相当于给自己的用户群提供一个"私有中转层"。

---

## 2. 请求的完整流程

```
客户端（Cursor/Codex CLI/自建应用）
    │
    │  POST https://longdaoai.cn/v1/chat/completions
    │  Authorization: Bearer user-api-key-xxx
    │
    ▼
Caddy（反向代理，处理 TLS、flush_interval）
    │
    │  HTTP → sub2api:8080（Docker 内网）
    │
    ▼
sub2api（鉴权 + 计费 + 路由）
    │  1. 验证 user-api-key-xxx 是否存在且有余额
    │  2. 根据模型名找到对应的 channel（上游配置）
    │  3. 记录请求开始，准备扣费
    │
    │  POST https://api.aipro.hk.cn/v1/chat/completions
    │  Authorization: Bearer upstream-api-key-yyy
    │  model: gpt-5.6-sol（原始模型名）
    │
    ▼
上游聚合中转站（aipro.hk.cn）
    │
    │  根据模型名路由到真正的提供商
    │
    ▼
OpenAI / Anthropic / Google
```

每个环节的分工：
- **Caddy**：处理 HTTPS、证书、流式 flush，不理解业务逻辑
- **sub2api**：业务核心，鉴权、计费、模型路由、用量记录
- **上游中转**：持有真实的 AI 提供商账号/额度，处理最后一公里

---

## 3. 上游（Upstream）vs 下游（Downstream）

这两个词在中转站语境里很容易混淆，用"水流方向"来记：

```
下游（你的用户）← [你的中转站] ← 上游（AI 提供商 / 聚合站）
```

- **上游**：你从哪里"买"算力。可以是 OpenAI 官方 API，也可以是聚合中转站（批发商）。
- **下游**：你的用户，他们调用你的接口，你给他们提供服务。

**商业逻辑**：聚合中转站（如 aipro.hk.cn）能以批发价拿到大量 token 额度，你以批发价从他们购买，再以零售价（稍高）卖给自己的用户，赚取差价，同时提供账号管理、限速等增值服务。

**内测阶段的简化**：龙道 AI 目前是封闭内测，不对外销售，所以"计费"只是用量统计，不涉及真实资金流转。管理员手动给用户账号充值余额，用于控制使用量。

---

## 4. 模型名映射（Model Mapping）

### 为什么需要映射

不同的上游中转站可能用不同的模型名。比如：

- 你的用户发请求用的是 `claude-3-5-sonnet`
- 但你的上游把这个模型注册为 `claude-3-5-sonnet-20241022`

如果直接透传，上游会返回"模型不存在"。Model Mapping 就是一个翻译字典：

```json
{
  "claude-3-5-sonnet": "claude-3-5-sonnet-20241022"
}
```

key 是用户请求的模型名，value 是转发给上游时实际使用的模型名。

### 这次遇到的大 Bug：Codex 模型规范化

**背景**：sub2api 为了支持 Codex CLI 的 OAuth 账号类型，在代码里加入了 `normalizeCodexModel` 函数，把一些非标准模型名自动规范化为 Codex 能识别的名字。

**问题**：这个规范化逻辑**对所有 OpenAI 类型账号无差别执行**，包括用 apikey 接入的第三方聚合站账号。

**具体现象**：
```
用户请求 model: "gpt-5.6-sol"
    ↓ normalizeCodexModel 介入
实际转发 model: "gpt-5.1"   ← 被改写了！
    ↓
上游聚合站收到 "gpt-5.1"，找不到对应供应商配置
    ↓
返回 404 / "model not found"
```

用户以为 `gpt-5.6-sol` 这个模型不可用，实际上是模型名被中间层悄悄改掉了。调试时非常难定位，因为 sub2api 的日志显示的是发出去的请求（已被改写），而不是用户原始请求。

**修复思路**：区分账号类型。

```
if (account.type === 'oauth') {
    // OAuth 账号走 ChatGPT 内部接口，需要规范化模型名
    model = normalizeCodexModel(model);
} else {
    // apikey 账号对接第三方兼容接口，保留原始模型名
    model = model;  // 不做任何改写
}
```

**后续**：上游 sub2api 项目在后来的版本中重构彻底移除了这个问题（升级后自然解决），但在旧版本上运行时需要手动打 patch。

---

## 5. `/v1/chat/completions` vs `/v1/responses`

OpenAI 有两个主要的对话接口：

### chat/completions（传统接口）

```http
POST /v1/chat/completions
{
  "model": "gpt-4o",
  "messages": [{"role": "user", "content": "hello"}],
  "stream": true
}
```

- 最通用，几乎所有 LLM 客户端和 SDK 都支持
- 返回 `data: {"choices":[{"delta":{"content":"Hi"}}]}` 格式的 SSE 流
- 适合绝大多数场景

### responses（新式接口）

```http
POST /v1/responses
{
  "model": "codex-mini-latest",
  "input": "write a hello world"
}
```

- OpenAI 2025 年推出的新接口，Codex CLI 使用这个端点
- 支持工具调用（function calling）的新格式
- 响应结构和 chat/completions 不同

**测试时两个端点都要覆盖的原因**：如果你的用户里有人用 Codex CLI，他们默认会打 `/v1/responses`；而大多数其他客户端（如 Cursor、OpenAI SDK）走 `/v1/chat/completions`。漏测其中一个，部分用户会遇到 404。

---

## 6. 账号类型：apikey vs OAuth

sub2api 支持两种 OpenAI 账号接入方式：

### apikey 账号

```
Authorization: Bearer sk-xxxxxxxxxxxx
```

- 标准 Bearer Token 认证
- 适合第三方兼容接口（聚合中转站、本地部署模型等）
- 模型名应该**原样透传**，不能做任何规范化

### OAuth 账号

- 走 ChatGPT 网页端的内部接口
- 需要先 OAuth 授权获取 access_token
- 这类账号接入的是 ChatGPT 原生接口，模型名有特定格式要求（如 `gpt-4o` 而不是 `gpt-4-turbo-preview`）
- 因此需要 `normalizeCodexModel` 把用户请求的模型名转换为官方接口认可的名字

**关键理解**：同一个模型名 `gpt-4o`，对 apikey 账号来说是直接透传给上游；对 OAuth 账号来说需要先规范化再发给官方接口。两条路径不能混用。

---

## 7. 用量记录与计费

### 请求 ID 去重（幂等结算）

网络不可靠，客户端可能重试同一个请求。如果每次重试都算一次计费，用户会被重复扣款。

解决方案：每个请求生成唯一 `request_id`（客户端生成或服务端生成后返回），结算时以 `request_id` 为幂等键：

```sql
INSERT INTO usage_records (request_id, user_id, tokens, cost)
VALUES (?, ?, ?, ?)
ON CONFLICT (request_id) DO NOTHING;  -- 重复请求不重复计费
```

### 余额扣减的事务性

计费的一致性要求：要么扣款成功且请求被记录，要么两者都不发生。用数据库事务保证：

```
BEGIN TRANSACTION
  1. 检查用户余额 >= 请求预估费用
  2. 扣减余额
  3. 记录请求
COMMIT
```

如果第 2 步和第 3 步之间服务崩溃，事务回滚，用户不会被扣款但请求也不计入记录。

### 封闭内测的简化方案

正式商业化需要对接支付宝/微信，流程复杂。内测阶段：

- **管理员手动充值**：在后台直接给用户账号加余额
- **兑换码**：生成一次性兑换码，用户自助兑换固定额度
- **不需要在线支付集成**：避免了支付 SDK、回调、对账等复杂度

这让内测可以在一周内跑起来，等产品验证后再接入支付。

---

*最后更新：2026-07-17*
