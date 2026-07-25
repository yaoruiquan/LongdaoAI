# 龙道 AI 部署知识库

> 本知识库基于龙道 AI Token 中转站的完整部署经历整理，记录了从零开始搭建、配置和升级过程中遇到的核心技术知识点。
> 每篇文档都包含「是什么、为什么、踩过什么坑」，适合回顾和深入学习。
>
> 创建日期：2026-07-17

---

## 📚 文档目录

### 🖥️ 服务器与网络

| 文件 | 内容 |
|------|------|
| [vps-and-network.md](./vps-and-network.md) | VPS 选型（日本 vs 香港/CN2 GIA/BGP）、域名 A 记录、HTTPS/TLS 原理、Caddy 反向代理、防火墙 |

### 🐳 容器与部署

| 文件 | 内容 |
|------|------|
| [docker-compose-basics.md](./docker-compose-basics.md) | 镜像 vs 容器、四服务架构、网络/数据卷/端口/健康检查/资源限制、常用命令、版本回滚 |
| [deployment-flow.md](./deployment-flow.md) | 标准发布序列（备份→迁移→部署→验证）、数据库迁移原理、幂等性、回滚策略、大版本升级经验 |

### 🔐 认证与安全

| 文件 | 内容 |
|------|------|
| [ssh-key-auth.md](./ssh-key-auth.md) | SSH 密钥认证原理、authorized_keys 权限、sshd_config 配置、~/.ssh/config 别名、Clash 代理干扰 |
| [github-auth.md](./github-auth.md) | PAT vs OAuth App Token、workflow scope、credential helper 优先级、HTTPS_PROXY 解决国内超时 |

### 🔀 代码管理

| 文件 | 内容 |
|------|------|
| [git-branch-management.md](./git-branch-management.md) | Fork 原理、版本 tag、git worktree、分叉历史挑战、rebase vs 重新移植的决策、分支策略 |

### 🤖 中转站架构

| 文件 | 内容 |
|------|------|
| [api-relay-architecture.md](./api-relay-architecture.md) | 请求完整链路、上下游关系、模型名映射、Codex 端点差异、apikey vs OAuth 账号、计费设计 |

---

## 🗺️ 快速定位

**我想了解某个具体问题，看哪篇？**

| 问题 | 看这里 |
|------|--------|
| 为什么选日本而不是香港？带宽怎么选？ | `vps-and-network.md` §2 |
| HTTPS 证书是怎么签发的？Caddy 做了什么？ | `vps-and-network.md` §4-5 |
| SSH 免密登录怎么配？为什么还是要密码？ | `ssh-key-auth.md` §5, §7 |
| Clash 为什么会截获 SSH 连接？ | `ssh-key-auth.md` §6, §7 |
| git push 为什么需要 workflow scope？ | `github-auth.md` §3-4 |
| gh CLI 的 token 和 PAT 有什么区别？ | `github-auth.md` §2 |
| Docker 容器停了数据会不会丢？ | `docker-compose-basics.md` §4 |
| `docker compose down -v` 有什么危险？ | `docker-compose-basics.md` §4 |
| 迁移文件为什么不能随便改？ | `deployment-flow.md` §3 |
| 136 改成 181 是怎么回事？ | `deployment-flow.md` §3 |
| 什么是幂等迁移？`IF NOT EXISTS` 有什么用？ | `deployment-flow.md` §3 |
| 为什么 gpt-5.6-sol 被改成了 gpt-5.1？ | `api-relay-architecture.md` §4 |
| `/responses` 和 `/chat/completions` 有什么区别？ | `api-relay-architecture.md` §5 |
| 2355 个 commit 的大版本升级怎么做的？ | `git-branch-management.md` §5, `deployment-flow.md` §5 |
| 怎么理解 `git worktree`？ | `git-branch-management.md` §3 |

---

## 📋 本项目已有的运维文档

| 文件 | 内容 |
|------|------|
| [../龙道AI服务器部署实录.md](../龙道AI服务器部署实录.md) | 14 步完整部署命令（可直接复用） |
| [../搭建AI中转站指南.md](../搭建AI中转站指南.md) | VPS/域名/上游选型、费用汇总、简单流程 |
| [../../deploy/production/OPS_README.md](../../deploy/production/OPS_README.md) | 运维脚本手册（backup/deploy/rollback） |
| [../../deploy/production/README.md](../../deploy/production/README.md) | 生产部署说明（compose/Caddy/环境变量） |
