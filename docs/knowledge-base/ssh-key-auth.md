# SSH 密钥认证

## 密钥认证 vs 密码认证

登录远程服务器有两种主流方式：密码认证和密钥认证。

| 对比项 | 密码认证 | 密钥认证 |
|--------|----------|----------|
| 安全性 | 密码可被暴力破解、钓鱼、泄露 | 私钥在本地，服务器永远拿不到 |
| 便利性 | 每次都要输入 | 配置一次，之后免密登录 |
| 自动化 | 脚本里写密码很危险 | 脚本天然支持，无需交互 |
| 抗中间人 | 弱（密码明文容易被截取） | 强（挑战-应答，不传输秘密） |

**密钥认证的本质**：服务器用你的公钥出一道"题"，只有持有私钥的人才能答对。整个过程中你的私钥从未离开本机。

---

## 非对称加密基础

用一个类比来理解：

> 你有一把**挂锁**（公钥）和对应的**钥匙**（私钥）。  
> 你可以把挂锁的复制品寄给任何人，但钥匙只有你有。  
> 别人用挂锁锁上一个盒子，只有你的钥匙才能打开。

- **公钥**（`.pub` 文件）：可以公开分发，放在服务器上
- **私钥**（无后缀文件）：严格保密，只存在本地，相当于物理钥匙

SSH 登录时的流程：
1. 客户端声明"我是 xxx，我有公钥 A"
2. 服务器查 `authorized_keys`，找到公钥 A，生成一个随机挑战
3. 服务器用公钥 A 加密这个挑战，发给客户端
4. 客户端用私钥解密，把答案发回服务器
5. 服务器确认答案正确 → 认证通过

这一过程中，**私钥的内容从未通过网络传输**。

---

## 关键文件说明

### 客户端（本机）

```
~/.ssh/
├── id_ed25519          # 私钥 —— 永远不要复制给任何人、任何服务
├── id_ed25519.pub      # 公钥 —— 可以随意分发，放到服务器、GitHub
├── known_hosts         # 已知服务器的指纹记录（防中间人攻击）
└── config              # SSH 客户端配置（别名、代理等）
```

生成密钥对（如果还没有）：
```bash
ssh-keygen -t ed25519 -C "your_email@example.com"
# 建议设置 passphrase，加一层本地保护
```

### 服务器端

```
~/.ssh/
└── authorized_keys     # 允许登录的公钥列表，一行一个
```

把本地公钥追加到服务器：
```bash
# 方法 1：ssh-copy-id（推荐）
ssh-copy-id -i ~/.ssh/id_ed25519.pub user@server

# 方法 2：手动追加
cat ~/.ssh/id_ed25519.pub | ssh user@server "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys"
```

---

## 权限要求（最常见的配置失败原因）

SSH 对权限非常敏感。权限设置错误时，服务器会**静默忽略** `authorized_keys`，不报任何有意义的错误，让你以为是别的问题。

### 正确权限

```bash
# 服务器上执行
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys

# 本机私钥
chmod 600 ~/.ssh/id_ed25519
```

### 为什么权限错了会导致认证失败？

SSH 的设计原则：如果 `authorized_keys` 对"其他用户"可写，那么任何人都可以往里面添加自己的公钥，认证就失去意义。服务器为了安全，直接拒绝读取权限过于宽松的配置文件，但**不会打印明显的错误提示**。

排查方法：加 `-vvv` 参数看详细日志：
```bash
ssh -vvv user@server
# 看输出中有没有 "Authentication refused: bad ownership or modes"
```

---

## 服务器端 SSH 配置（`/etc/ssh/sshd_config`）

服务器的 SSH 服务由 `sshd` 守护进程控制，主配置文件是 `/etc/ssh/sshd_config`。

### 关键配置项

```ini
# 开启公钥认证（默认 yes，但有些镜像会关掉）
PubkeyAuthentication yes

# 是否允许密码登录（安全加固时设为 no）
PasswordAuthentication yes

# 指定 authorized_keys 文件的路径（一般不用改）
AuthorizedKeysFile .ssh/authorized_keys

# 是否允许 root 直接 SSH 登录（建议 no）
PermitRootLogin prohibit-password
```

### 重要：`PubkeyAuthentication no` 是本次踩的坑

部分云服务商的镜像（或安全基线模板）会把 `PubkeyAuthentication` 设为 `no`，导致密钥完全无效。你把公钥放好、权限也对，就是登不上去。

检查方法：
```bash
grep -i PubkeyAuthentication /etc/ssh/sshd_config
```

修改后必须重启 sshd 服务才生效：
```bash
systemctl restart sshd
# 或者
service ssh restart
```

**注意**：重启前先保留一个已登录的 shell 会话，防止配置有误被锁在外面。

---

## `~/.ssh/config` 配置文件

`config` 文件是 SSH 客户端的"快捷方式"配置，让你用短命令代替长参数。

### 基本格式

```
Host <别名>
    HostName <真实IP或域名>
    User <用户名>
    Port <端口，默认22>
    IdentityFile <私钥路径>
    ProxyCommand <代理命令，或 none>
```

### 实际案例：本次 longdao 服务器配置

```
Host longdao
    HostName 203.0.113.42
    User root
    IdentityFile ~/.ssh/id_ed25519
    ProxyCommand none
```

配置后，`ssh longdao` 等同于 `ssh -i ~/.ssh/id_ed25519 root@203.0.113.42`。

### 常用字段说明

| 字段 | 说明 | 示例 |
|------|------|------|
| `Host` | 别名，用于命令行的短名 | `longdao` |
| `HostName` | 真实的 IP 或域名 | `203.0.113.42` |
| `User` | 登录用户名 | `root` |
| `Port` | SSH 端口（默认 22） | `2222` |
| `IdentityFile` | 指定用哪个私钥 | `~/.ssh/id_ed25519` |
| `ProxyCommand` | 连接代理命令，`none` 表示不走代理 | `none` |

### `ProxyCommand none`：为什么需要它？

在开启了 Clash 等代理软件的 Mac 上，系统代理规则可能包含通配符，把 SSH 流量也路由到代理。这会导致连接被截断或握手失败。

加上 `ProxyCommand none` 显式告诉 SSH 客户端：**跳过任何全局代理设置，直接连接目标**。

```
# 没有这行 → Clash 可能截获 → 连接超时或握手失败
# 有这行   → SSH 直连目标 IP
ProxyCommand none
```

---

## 常见问题总结

### 1. 服务器关闭了 `PubkeyAuthentication`

**症状**：公钥放好了，权限也对，还是要求输密码，或者直接 `Permission denied`。  
**排查**：`grep PubkeyAuthentication /etc/ssh/sshd_config`  
**修复**：改为 `yes`，然后 `systemctl restart sshd`

### 2. `authorized_keys` 权限不对

**症状**：密码登录可以，密钥登录不行，`-vvv` 输出里有 `bad ownership or modes`。  
**修复**：
```bash
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
```

### 3. 代理软件（Clash）截获 SSH 连接

**症状**：直接 `ssh user@ip` 超时，但网络是通的；或者 ping 通但 SSH 握手失败。  
**修复**：在 `~/.ssh/config` 里加 `ProxyCommand none`，或者临时关闭代理。

### 4. zsh 中 heredoc 语法注意点

在 zsh 里用 heredoc 向远程写文件时，要用引号把结束符括起来，防止本地变量展开：

```bash
# 错误：$变量会在本地展开
ssh server "cat >> ~/.ssh/authorized_keys" << EOF
$(cat ~/.ssh/id_ed25519.pub)
EOF

# 正确：用单引号包住结束符，防止本地展开
ssh server "cat >> ~/.ssh/authorized_keys" << 'EOF'
ssh-ed25519 AAAA... your_comment
EOF
```

---

最后更新：2026-07-17

