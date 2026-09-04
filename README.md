# 龙道 AI - 算力中转站

企业级 AI 算力中转与管理平台，提供统一的 API 接口访问多家上游 AI 服务。

## 核心功能

### 🔌 多平台接入
- **OpenAI** - GPT-4、GPT-4o、GPT-5 系列、DALL-E 图像生成
- **Anthropic Claude** - Claude 3.5/3 Opus/Sonnet/Haiku
- **Google Gemini** - Gemini 1.5 Pro/Flash
- **国产大模型** - 智谱 GLM、Deepseek、Kimi、Grok 等

### 💼 企业级特性
- **用户管理** - 多租户支持，独立账号体系
- **分组管理** - 灵活的用户分组与权限控制
- **账号池** - 多上游账号自动负载均衡与故障转移
- **代理管理** - 支持多代理配置，按需路由
- **用量统计** - 实时用量追踪、成本核算
- **计费系统** - 灵活的定价策略、余额管理、充值系统

### 🛡️ 高可用设计
- **蓝绿部署** - 零停机更新
- **健康检查** - 自动故障检测与恢复
- **限流保护** - 多层级限流策略
- **缓存优化** - Redis 缓存加速
- **日志审计** - 完整的操作日志追踪

### 🎯 便捷接入
- **OpenAI 兼容** - 无缝替换 OpenAI API endpoint
- **多种认证** - API Key、OAuth 2.0
- **流式输出** - 支持 Server-Sent Events (SSE)
- **图像生成** - 完整的 DALL-E API 支持
- **批量任务** - 异步图像生成任务队列

## 技术栈

- **后端**: Go 1.26 + Gin + Ent ORM
- **前端**: Vue 3 + Vite + Element Plus
- **数据库**: PostgreSQL 18
- **缓存**: Redis 7
- **部署**: Docker + Docker Compose

## 快速开始

### 环境要求
- Docker 20.10+
- Docker Compose 2.0+

### 部署

```bash
# 1. 克隆仓库
git clone https://github.com/tokeys/longdaoai.git
cd longdaoai

# 2. 配置环境变量
cp deploy/production/.env.example deploy/production/.env
vim deploy/production/.env  # 修改数据库密码等敏感信息

# 3. 启动服务
cd deploy/production
./deploy.sh

# 4. 访问管理后台
# 浏览器打开 http://your-domain
# 首次访问会进入初始化向导
```

### 首次配置

1. **初始化向导** - 设置管理员账号和数据库连接
2. **添加上游账号** - 在"账号管理"中添加上游 API 凭证
3. **创建用户分组** - 配置用户可访问的模型和账号池
4. **创建用户** - 分配用户到对应分组
5. **生成 API Key** - 为用户生成访问密钥

## API 使用示例

### OpenAI 兼容接口

```bash
curl https://your-domain/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'
```

### 图像生成

```bash
curl https://your-domain/v1/images/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "dall-e-3",
    "prompt": "A beautiful sunset over mountains",
    "size": "1024x1024"
  }'
```

## 架构特性

- **零停机部署** - 蓝绿切换，无缝更新
- **自动健康检查** - 定时探测上游账号可用性
- **智能调度** - 基于优先级、并发数、限流状态的智能选号
- **故障转移** - 上游失败自动切换备用账号
- **安全加固** - 密钥加密存储、HTTPS 强制、安全响应头

## 运维管理

- **日志查看**: `docker logs -f longdao-sub2api-blue`
- **数据备份**: 自动定时备份（PostgreSQL + Redis）
- **监控告警**: 集成 OPS 监控模块
- **故障排查**: 内置诊断工具

## 安全建议

1. **修改默认密码** - 务必修改 PostgreSQL、Redis 默认密码
2. **配置 HTTPS** - 生产环境必须启用 SSL/TLS
3. **限制访问** - 使用防火墙限制管理端口访问
4. **定期备份** - 配置自动备份策略
5. **密钥轮换** - 定期更换上游 API Key 和用户密钥

## 许可证

本项目基于原始 [sub2api](https://github.com/Wei-Shaw/sub2api) 项目开发。

---

**龙道 AI** - 让算力管理更简单
