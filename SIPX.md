# sipx — SIP 外呼快捷工具使用说明

`sipx` 是 [sipexer](https://github.com/miconda/sipexer) 的包装命令行：把常用的 SIP
话机配置（注册号码、服务地址、鉴权、呼叫参数）保存成**配置档（profile）**，并记录
**外呼历史**，让重复外呼变成一条短命令。

`sipx` 本身不改 sipexer 引擎，而是找到 `sipexer` 二进制后以子进程方式执行；
`sipexer` 的行为、参数、退出码与原版完全一致。

```
┌────────┐  dial   ┌──────────┐   子进程+参数   ┌──────────┐  SIP   ┌──────┐
│  sipx  ├────────►│ profiles │───────────────►│ sipexer  ├───────►│ PBX  │
│        │         │ history  │                │ (引擎)   │◄───────┤      │
└────────┘         └──────────┘                └──────────┘        └──────┘
```

---

## 1. 安装

解压发行包（`sipexer-sipx-<版本>-<系统>-<架构>.tar.gz`）后：

```
tar xzf sipexer-sipx-2.0.0-linux-amd64.tar.gz
sudo cp bin/sipexer bin/sipx /usr/local/bin/     # 或拷贝到任意 PATH 目录
```

两个二进制**放在同一目录**即可，`sipx` 会自动找到旁边的 `sipexer`。

### 引擎查找顺序

| 优先级 | 方式 |
|---|---|
| 1 | `--engine /path/to/sipexer` 全局参数 |
| 2 | 环境变量 `SIPEXER_BIN` |
| 3 | `sipx` 同目录下的 `sipexer` |
| 4 | `PATH` 中的 `sipexer` |

验证：

```
sipx doctor        # 显示找到的引擎路径与版本
```

### 配置文件位置

- `~/.config/sipexer/profiles.json`（配置档）
- `~/.config/sipexer/history.json`（外呼历史）
- 可用 `--config-dir <dir>` 或环境变量 `SIPEXER_CONFIG_DIR` 改到别处
  （Linux 也支持 `XDG_CONFIG_HOME`；macOS 默认 `~/.config`，可用环境变量覆盖）

> 密码/HA1 在文件里做了混淆（防君子不防小人），**不是加密**；请用文件权限保护该目录。

---

## 2. 快速上手

```
# 1) 建一个配置档：注册号码 + 服务器 + 鉴权 + 常用呼叫参数
sipx phone add work --auth-user 18800012001 --auth-password '123456' \
    --number 18800012001 \
    --server udp://10.1.106.181:38896 \
    --register-first --method INVITE --set-user --contact-build \
    --sw 60000 --vl 3

# 2) 设为当前配置
sipx phone default work

# 3) 拨号（等价于原来那一大串 sipexer 参数）
sipx dial 320196120001
```

## 3. 命令一览

| 命令 | 作用 |
|---|---|
| `sipx phone add <名> [flags]` | 新建配置档 |
| `sipx phone edit <名> [flags]` | 修改（未给的字段保留） |
| `sipx phone list` | 列出全部（默认项带 `*`，密码不显示） |
| `sipx phone show <名>` | 查看详情 |
| `sipx phone rm <名>` | 删除 |
| `sipx phone default [名]` | 设置/查看当前配置（别名 `current`） |
| `sipx server add <profile> <addr> [-d]` | 给配置档添加服务地址（`-d` 设默认） |
| `sipx server rm <profile> <addr>` | 删除服务地址 |
| `sipx server list <profile>` | 列出服务地址 |
| `sipx dial [profile] [target]` | 拨号 |
| `sipx history` / `h` | 外呼历史（最新在前） |
| `sipx history alias <@N> <名>` | 给某条历史起别名 |
| `sipx history clear` | 清空历史（会先确认） |
| `sipx doctor` | 检查引擎 |
| `sipx version` / `--version` | 版本 |

## 4. dial：拨号

```
sipx dial work 320196120001      # 指定配置 + 号码
sipx dial 320196120001           # 用当前配置（phone default 设的那个）
sipx dial default 320196120001   # 同上，显式写法
sipx dial desk                   # 号码别名（配置档里登记的号码）
sipx dial @0                     # 重呼最近一次
sipx dial boss                   # 呼历史条目的别名
sipx dial                        # 终端下进入选择菜单：列历史让你选
sipx dial -s udp://其他:5060 1001    # 临时换服务器
sipx dial --sw 2000 1001             # 临时改参数，覆盖配置里的默认
sipx dial work 1001 -- -vl 3 -co     # “--” 后面原样透传给 sipexer
```

目标解析顺序：`@N 索引` → 配置档号码别名 → 历史别名 → 当作裸号码/URI。

**退出码**与 sipexer 一致：成功通话结束时引擎以**最终 SIP 响应码**退出
（例如 `200`），并非错误码。

### 配置档可保存的呼叫参数

| sipx 参数 | 对应 sipexer | 说明 |
|---|---|---|
| `--method INVITE/REGISTER/...` | `-i` / `-r` ... | 方法 |
| `--register-first` | `--register-first` | 先注册再呼叫 |
| `--set-user` | `-su` | R-URI user 取被叫 |
| `--contact-build` | `-cb` | 用本地地址构造 Contact |
| `--sw N` | `-sw N` | 应答后等待 N 毫秒再挂断（见 §6） |
| `--cd/--rt N` | `-cd/-rt` | 通话/振铃时长（self-call 模式） |
| `--timeout/--timeout-connect/--timeout-write` | 同名 | 各类超时 |
| `--vl N` | `-vl` | 日志级别 0..3 |
| `--co/--com` | `-co/-com` | 彩色输出/消息 |
| `--ti` | `-ti` | 跳过 TLS 校验 |
| `--ex/--ua/--cu/--ct/--mb` | 同名 | Expires/UA/Contact/类型/消息体 |
| `--xh 'Name: value'` | `-xh` | 附加 SIP 头（可重复） |
| `--extra-dial-flag '...'` | 任意 | 兜底：原样附加给引擎的参数 |

拨号时同名参数优先于配置里的值。

### 号码与服务地址

- 一个配置档可登记**多个号码**，`--number '别名=user@domain'`（可重复）；
  别名可直接用来拨号：`sipx dial alice`。
- 一个配置档可配**多个服务地址**（`proto://host:port`），
  默认项用 `-d/--default` 标记；`sipx dial -s <addr>` 临时覆盖。

## 5. history：外呼历史

- 每次 dial 记录：时间、配置档、服务器、主叫号码、目标号码。
- **同一（配置档,主叫,目标）只保留一条**：再呼会更新时间并置顶，别名保留。
- 默认最多 100 条，超出淘汰最旧。
- `sipx history alias 0 boss` 起别名后即可 `sipx dial boss` 一键重呼。
- 列表里 `@0` = 最新，`@1` 次新，以此类推。

## 6. 挂机与 BYE（重要）

sipexer 引擎在 INVITE 成功后会等 `--sw`（session wait）指定的毫秒数，然后**自动
发送 BYE 挂机**。所以：

- 正常挂机：把 `--sw` 设为期望的最长通话时间，时间一到引擎自己发 BYE，`sipx`
  透传引擎退出码。
- **Ctrl-C 会直接杀死引擎进程，不发 BYE**（引擎没有信号处理）。对方网关需要
  等待超时才会释放会话。测试中如需提前挂机，用更小的 `--sw` 重新拨号，或让
  对端挂断。

## 7. 与原版 sipexer 的关系

- `sipx` 只加壳不改核：`sipexer` 二进制原样使用，原有脚本/用法完全不受影响。
- 需要引擎的任意冷门能力时：
  `sipx dial <profile> <target> -- <任意 sipexer 参数>` 直接透传，
  或者干脆继续用 `sipexer ...` 原始命令。

## 8. 故障排查

| 现象 | 处理 |
|---|---|
| `sipx: sipexer engine not found` | 把 `sipexer` 与 `sipx` 放同目录，或 `--engine` / `SIPEXER_BIN` 指定 |
| 拨号秒退、无鉴权 | 检查 `--auth-user/--auth-password`；HA1 用 `--ha1` |
| 想换配置目录/迁移 | `--config-dir` 或 `SIPEXER_CONFIG_DIR` |
| 不知道引擎在哪 | `sipx doctor` |
| 参数不认识 | `sipx dial --help`；`--` 透传兜底 |
```