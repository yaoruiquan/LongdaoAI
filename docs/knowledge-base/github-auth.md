# GitHub 认证：PAT、OAuth Token 与 workflow scope

## 三种 GitHub 认证方式

| 方式 | 状态 | 适用场景 |
|------|------|----------|
| HTTPS 密码 | **已废弃**（2021年8月起） | 不再支持 |
| PAT（Personal Access Token） | 推荐 | 脚本、CI、需要精确控制权限 |
| SSH Key | 推荐 | 日常 git push/pull，无需过期管理 |

HTTPS 密码认证已被 GitHub 彻底关闭。现在用 HTTPS 协议推代码，必须用 PAT（或 OAuth Token）作为密码。

---

## PAT 与 OAuth App Token 的区别

这是最容易混淆的地方，也是很多人遇到"明明有 token 却推不了代码"的根源。

### PAT（Personal Access Token）

- **你**在 GitHub Settings → Developer settings → Personal access tokens 手动创建
- scope 完全由你决定，创建时可以勾选 `repo`、`workflow` 等任意权限
- 有两种类型：
  - **Fine-grained PAT**：精确到仓库级别的权限控制（推荐）
  - **Classic PAT**：老式的宽泛 scope 控制

### `gh` CLI 生成的 OAuth App Token

- 运行 `gh auth login` 后，`gh` 会用 GitHub OAuth App 的授权流程拿到一个 token
- **scope 由 GitHub CLI 这个 OAuth App 决定**，通常包含 `repo`、`read:org` 等，但**不包含 `workflow`**
- 这个 token 存在 macOS Keychain 或 `~/.config/gh/hosts.yml` 里

### 关键区别

```
PAT        你创建 → 你控制 scope → 可以有 workflow
OAuth Token  gh 申请 → OAuth App 控制 scope → 通常没有 workflow
```

**实际案例**：你用 `gh auth login` 登录了，`gh` 工作正常，`gh repo view` 也没问题，但一旦推送包含 `.github/workflows/` 文件的 commit，就报 `refusing to allow a OAuth App to create or update workflow`。原因就是 OAuth Token 没有 `workflow` scope。

---

## Scope（权限范围）

| Scope | 含义 |
|-------|------|
| `repo` | 读写仓库内容（代码、issue、PR） |
| `workflow` | 创建或修改 `.github/workflows/` 下的文件 |
| `read:org` | 读取组织信息 |
| `admin:repo_hook` | 管理 webhook |
| `delete_repo` | 删除仓库（高危，一般不要给） |

**为什么推送 CI workflow 文件需要 `workflow` scope？**

GitHub 把 `.github/workflows/` 下的文件当作特殊敏感资源——修改 workflow 文件相当于修改 CI/CD 流水线，可以执行任意代码。因此 GitHub 要求推送方必须有显式的 `workflow` 授权，即使你有完整的 `repo` 权限也不够。

---

## git credential helper 的优先级

这是本次部署实际踩到的坑，值得记录清楚。

### 配置层级

```
credential.https://github.com.helper   （per-host，优先级最高）
credential.helper                       （全局，优先级次之）
/etc/gitconfig 里的 credential.helper  （系统级，优先级最低）
```

### `gh auth setup-git` 做了什么

运行这个命令后，它会在 `~/.gitconfig` 里写入：

```ini
[credential "https://github.com"]
    helper = !/usr/local/bin/gh auth git-credential
```

这是 per-host 配置，**优先级高于** macOS 的 `osxkeychain`（全局）。所以即使你在 Keychain 里存了 PAT，git push 时也会走 `gh auth git-credential`，拿到的是 OAuth Token，从而触发 workflow scope 限制。

### 如何确认当前使用的是哪个 helper

```bash
git config --list | grep credential
# 看看有没有 per-host 配置覆盖了全局
```

### 临时绕过 per-host helper 的技巧

当你确认问题是 per-host 的 `gh auth git-credential` 在作怪，可以用 `-c` 临时清除它，让 git 降级到全局的 `osxkeychain`（里面存着你的 PAT）：

```bash
git -c 'credential.https://github.com.helper=' push origin main
```

这个空字符串会"取消"per-host 配置，git 于是向上查找全局配置，找到 `osxkeychain`，从而拿到 PAT（有 `workflow` scope）来完成推送。

这只是临时绕过，不会修改 `~/.gitconfig`。长期方案见下方"最佳实践"。

---

## device flow 认证（`gh auth refresh`）

`gh auth login` 和 `gh auth refresh` 使用 OAuth Device Flow：在命令行打印一个 URL 和验证码，让你在浏览器里授权。

### 为什么在国内经常超时？

Device Flow 的整个流程涉及多次对 `github.com` 的 HTTPS 请求。国内网络直连 `github.com` 延迟高且不稳定，常见的失败表现：

- 命令卡住，一直等待
- 打开了浏览器验证码页面，但 CLI 那边等到超时

### 解决方案：让 `gh` 走代理

```bash
# Clash 默认混合代理端口是 7897（或 7890，按实际配置）
HTTPS_PROXY=http://127.0.0.1:7897 gh auth login
HTTPS_PROXY=http://127.0.0.1:7897 gh auth refresh -s workflow
```

`gh` 会读取 `HTTPS_PROXY` 环境变量，把所有 HTTPS 请求走代理，解决连接超时问题。

---

## 最佳实践建议

### 1. 维护一个长期 PAT

在 GitHub → Settings → Developer settings → Personal access tokens → Fine-grained tokens 创建一个 PAT：

- Scope 至少包含：`repo`（Contents: Read & Write）和 `workflow`
- 设置合理的过期时间（建议 90 天或 1 年）
- 把它存入 macOS Keychain（`osxkeychain`）：
  ```bash
  git config --global credential.helper osxkeychain
  # 下次 git push 时输入用户名 + PAT 作为密码，会自动保存
  ```

### 2. 避免 per-host 被 `gh` 覆盖

如果你需要 `gh` CLI 正常工作，同时又要用 PAT 推代码，可以**不运行** `gh auth setup-git`，或者在运行后手动检查 `~/.gitconfig`，删除 `[credential "https://github.com"]` 这一节。

### 3. 定期 revoke 过期或泄露的 token

- GitHub Settings → Developer settings → Personal access tokens → 找到对应 token → Revoke
- 如果 token 出现在 git 历史里，除了 revoke 还要清理历史（`git filter-repo`）

### 4. 永远不要在终端历史里粘贴 token 明文

```bash
# 危险：token 会出现在 ~/.zsh_history
git remote set-url origin https://ghp_xxxxxxxxxxxx@github.com/user/repo

# 安全：交互式输入，不留历史
git push  # 让 credential helper 来处理
```

如果不慎在 shell 里打印了 token，立刻去 GitHub 撤销它。

---

最后更新：2026-07-17

