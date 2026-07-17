# Git 分支、Fork 与大版本升级

> 面向有 Git 基础但没有维护过长期 Fork 项目的开发者。

---

## 1. Fork 是什么

Fork 在 GitHub 上的字面意思是"叉子"，就是把别人的仓库复制一份到你名下，在此基础上做自己的版本。

```
upstream/sub2api（原作者）
    │
    └── fork ──▶ longdao/sub2api（你的版本）
                    │
                    ├── 原作者的所有代码
                    └── 你的定制改动（Caddy 配置、模型映射、部署脚本等）
```

**和 upstream 的关系**：

- upstream 持续迭代（修 bug、加功能），你的 fork 不会自动同步
- 你需要主动拉取 upstream 的更新，合并到自己的分支
- 你的定制改动和 upstream 的新功能可能冲突，需要手动解决

Fork 的核心挑战是**长期维护**：一开始分叉量少，合并容易；随着时间推移，双方都在改动，合并越来越复杂。

---

## 2. 版本 Tag：不可变的快照

### 为什么用日期版本号

sub2api 使用 `v2026.07.17-1` 这样的版本格式（CalVer，日历版本）：

- `2026.07.17`：日期，一眼知道这是哪天的版本
- `-1`：同一天的第几个版本，修复紧急 bug 时会出 `-2`、`-3`

相比 `latest` 标签，固定版本号的优势：
- **`latest` 是可变的**：今天部署的 `latest` 和明天的 `latest` 可能是不同的代码
- **版本号是不可变的**：`v2026.07.17-1` 永远指向同一个 commit，便于回滚和追溯

### 常用 tag 操作

```bash
# 查看所有 tag（按版本排序）
git tag --sort=-version:refname | head -10

# 查看某个 tag 对应的 commit
git show v2026.07.15-1 --stat

# 切换到某个 tag（分离 HEAD 状态，只读）
git checkout v2026.07.15-1

# 回到分支
git checkout main

# 基于某个 tag 创建分支
git checkout -b hotfix/xxx v2026.07.15-1
```

---

## 3. git worktree：隔离工作区

### 问题场景

你的生产服务器运行着 `main` 分支的代码。你想测试一个大版本升级（比如 upstream 的新架构），但：

1. 直接在 `main` 分支改动太危险，一旦出错服务就停了
2. `git checkout` 切换分支会改变工作目录，Docker Compose 的挂载文件可能受影响

### worktree 的解法

`git worktree` 允许你在**同一个 git 仓库**里同时检出多个分支到**不同目录**：

```bash
# 在 ../sub2api-upgrade 目录检出 longdao-upgrade 分支
git worktree add ../sub2api-upgrade longdao-upgrade

# 现在你有两个目录：
# /opt/sub2api           ← main 分支，生产正在跑
# /opt/sub2api-upgrade   ← longdao-upgrade 分支，用来测试升级
```

两个目录共用同一个 `.git` 数据库，分支互相独立，互不影响。

### worktree vs 普通 checkout 的区别

| 操作 | git checkout | git worktree |
|------|-------------|--------------|
| 工作目录 | 只有一个 | 多个并存 |
| 切分支影响生产 | 是 | 否 |
| 适合场景 | 日常开发 | 升级测试、并行开发 |

---

## 4. 分叉历史的挑战

### 越来越难合并的根本原因

当你 fork 了 upstream 并持续改动，历史树会长这样：

```
upstream: A → B → C → D → E → F → G（共 2000+ commits）
                   ↑
你的 fork:         X → Y → Z（你的改动）
```

upstream 和你的 fork 在 C 之后"分叉"了。时间越长：
- upstream 新增的 commit 越多（D、E、F、G...）
- 你的 fork 积累的 commit 也越多（X、Y、Z...）
- 双方改动的文件越来越多，冲突概率越大

### 本次经历：2355 commits 的分叉

旧的 longdao fork 和 upstream/main 已经分叉了 **2355 commits**。这意味着：

- `git log --oneline upstream/main ^HEAD` 显示 upstream 领先 2355 个提交
- 大量文件被双方各自改动过
- 如果用 `git merge upstream/main`，会产生数百个冲突文件

这种情况下，传统的"拉 upstream 再合并"策略已经不现实——冲突太多，逐一解决的工作量远超价值。

---

## 5. 两种升级策略对比

### 策略一：rebase

把你的 commits 逐个"重放"到最新 upstream 之上：

```bash
git fetch upstream
git rebase upstream/main
# 解决每个 commit 的冲突
# 重复直到 rebase 完成
```

**优点**：历史干净，你的改动都变成基于最新 upstream 的 commits。
**缺点**：commits 越多，冲突解决越多。如果有 50 个 commits，可能每个都要处理冲突。

**适用场景**：你的定制改动不多（< 10 个 commits），和 upstream 分叉时间短。

### 策略二：全新起步 + 移植（本次选择）

以 upstream 最新版本为基础，重新开一个分支，然后把你的定制改动**选择性地搬过来**：

```bash
# 1. 基于 upstream 最新 tag 创建新分支
git checkout -b longdao-upgrade v2026.07.17-1

# 2. 找出你的定制改动文件列表
git diff v2026.07.15-1 HEAD --name-only

# 3. 逐文件检查，手动移植真正需要的改动
git show HEAD:path/to/file | diff - path/to/file

# 4. 测试验证通过后，force-push 更新 main
```

**优点**：从干净的起点出发，只搬"真正需要的改动"，冲突更可控。
**缺点**：需要手动梳理哪些改动要保留，容易遗漏。

**适用场景**：commits 数量多、分叉时间长、跨越大版本（架构有变化）。

### 决策树

```
分叉 commits < 20 个？
├── 是 → 尝试 rebase，冲突多再换方案
└── 否 → 评估定制改动范围
         ├── 只改了少数文件（< 10 个）→ 手动移植
         └── 改动广泛 → 全新起步 + 移植
```

---

## 6. 分支策略

### 升级工作分支

升级期间不要直接在 main 上工作。用独立分支：

```bash
git checkout -b longdao-upgrade
# 在这里做升级、测试、修复
# 确认没问题后再合并/替换 main
```

好处：
- main 始终是"已知可用"的状态
- 升级分支出问题直接丢弃，无损

### force-push 的时机和风险

正常情况下 `git push --force` 是危险操作：它会覆盖远端历史，团队其他人的本地分支会和远端不一致。

**可以用 force-push 的情况**：
- 分支只有你一个人在用（如 `longdao-upgrade`）
- 升级完成后用新历史替换旧的 main（团队已同步）
- 修正刚刚推上去的错误 commit（还没人拉取）

**绝对不行的情况**：
- 团队共享的 main/develop 分支有其他人在基于它工作
- 生产 CI/CD 依赖这个分支的特定 commit

**用 `--force-with-lease` 代替 `--force`**：它会先检查远端的当前状态，如果和你预期不符（说明有人推了新 commit）会拒绝操作，更安全。

```bash
git push --force-with-lease origin main
```

### 服务器与 GitHub 保持一致

生产服务器上的代码应该始终是 GitHub 上某个特定 tag 或 commit 的精确副本。

```bash
# 服务器部署时明确指定版本
git fetch origin
git checkout v2026.07.17-1  # 固定 tag，不用 main 分支名

# 或者
git reset --hard origin/main  # 和远端 main 对齐（谨慎使用）
```

如果服务器上有手动改动（直接在服务器上 vim 改了配置），会导致：
1. 下次部署时和 GitHub 不一致，自动部署失败
2. 无法回滚（本地改动不在 git 历史里）

**原则**：服务器上不做任何手动改动，所有配置变更都通过 git 提交。

---

## 7. 常用 Git 操作

### 查看上游进展

```bash
git remote add upstream https://github.com/original/sub2api.git
git fetch upstream

# 看 upstream 最近 5 个 commit
git log upstream/main --oneline -5

# 看 upstream 比我们领先多少
git log HEAD..upstream/main --oneline | wc -l
```

### 看我们改了哪些文件

```bash
# 相对于某个 commit，我们改了哪些文件
git diff 0507852a HEAD --name-only

# 某个文件改了什么
git diff 0507852a HEAD -- src/gateway/relay.go

# 我们有哪些 commit 是 upstream 没有的
git log upstream/main..HEAD --oneline
```

### 挑选特定 commit 移植（cherry-pick）

```bash
# 把某个 commit 的改动应用到当前分支
git cherry-pick abc1234

# 移植但不立即提交（先检查）
git cherry-pick --no-commit abc1234

# 移植一段范围的 commits
git cherry-pick abc1234^..def5678
```

### 查看某文件在某版本的内容

```bash
# 查看 v2026.07.15-1 版本的某文件
git show v2026.07.15-1:docker-compose.yml

# 对比两个版本的某文件
git diff v2026.07.15-1 v2026.07.17-1 -- docker-compose.yml
```

### force-push 的正确姿势

```bash
# 确认当前状态
git log --oneline -5
git status

# 确认远端状态（没有意外的新 commit）
git fetch origin
git log origin/main --oneline -3

# 执行 force push（带保护）
git push --force-with-lease origin main
```

---

*最后更新：2026-07-17*
