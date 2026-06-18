# bitExchange — 第二阶段公共网络互联能力

日期：2026-06-10

## 1. 范围

第二阶段在 Phase 1 Go CLI / Go server 基础上，把“局域网直连 MVP”扩展为“公网辅助寻址 + 多路径连接选择”的 CLI 可验证版本。本阶段不引入桌面/Android/Web UI。

新增能力：

- 公网信令服务：在线注册、私有候选地址收集和交换
- 加密端到端信令转发
- 客户端多路径连接管理器的探测与回退
- 可选中继服务：本阶段仅支持文本消息和小文件

## 2. 连接路径优先级

客户端发送数据时按以下顺序尝试：

1. 同一子网直连
2. 通过公网服务器交换双方私有候选地址后的私有网络直连
3. 公网信令辅助的 P2P 打洞直连
4. 可选公网中继
5. 失败提示

每次传输均选择当前最佳路径。弱于上次的路径不自动选中，除非连接中断后重新尝试。公网中继是兜底方案，用户可以选择关闭；关闭后 P2P 失败即提示无法连接。

## 3. 服务端信令模型

公网服务器在本阶段作为“在线状态 + 候选地址 + 加密信令 + 小规模中继”的中心节点，不保存聊天记录，也不保存文件。客户端上线时向服务端注册：

- device_id
- device_name
- public_key fingerprint
- 监听端口
- 私有网络 IP/端口候选列表
- 可见的远端地址信息
- 是否允许中继
- 中继上限大小配置

服务端维护内存态在线表：`device_id → session`，每个 session 包含心跳时间、候选地址、信令通道和 relay 能力。服务端仅允许已配对或同一可信设备组内的设备相互查询在线状态；候选地址和连接协商消息以端到端加密形式转发，不做内容解析。

信令事件包括：

- **online_register**：客户端上线并提交候选地址。
- **private_candidates_offer**：服务端将双方私有 IP/端口候选地址互发。
- **p2p_signal**：客户端间交换 P2P 打洞所需的加密协商消息。
- **relay_request / relay_ready**：直连失败后申请中继通道；本阶段仅支持文本和不超过 `relay_max_bytes` 的小文件。

心跳超时后服务端从在线表中移除该设备。服务端重启后在线表清空，客户端自动重连并重新注册。

## 4. 客户端路径选择与探测流程

发送文本或文件时，客户端统一通过“连接路径选择器”处理。

### 4.1 同子网直连

使用原有 mDNS/UDP 广播发现目标，尝试 TCP 探测其监听端口。成功则直接复用 Phase 1 直连传输。

### 4.2 私有候选地址探测

同上一步不通时，连上公网服务器注册并获取目标端的私有候选地址。本机对每一个候选地址做并发送探测；目标端也可反向探测本机候选地址。任意方向成功即标记为“私有直连”，后续传输走该路径。

### 4.3 公网 P2P 打洞

私有候选探测仍失败时，通过服务端加密转发 P2P 协商消息。本阶段先实现可替换接口，默认用 TCP hole-punch / connection negotiation 模型，预留后续 QUIC/UDP 升级点。成功后走 P2P 直连。

### 4.4 可选中继

若用户允许中继，且以上路径均不可用，客户端发起 relay_request。本阶段只支持文本和不大于 `relay_max_bytes` 的文件。超出限制时直接返回错误，不做自动回退或拆分。如果用户关闭中继，直接报告明确失败原因。

客户端缓存最近一次成功路径，下一次对同一设备发送时优先尝试该路径；如果失败则从优先级最高路径开始完整重试。

## 5. 安全与端到端信令加密

本阶段确保公网服务器不能伪造设备身份，也不能读取敏感信令内容。

- 设备身份沿用 Phase 1 的长期公私钥、设备 ID 和指纹；可信设备存储于本机 `trusted-peers.json`。
- 客户端与公网服务器通过 HTTPS/WSS 通信。
- A 发给 B 的候选地址确认、P2P 协商和中继请求等敏感信令，在客户端用 B 的公钥加密并由 A 签名后才交给服务器转发。
- B 接收时验证 A 的设备指纹和签名；不匹配或不在可信列表内均拒绝处理。
- 每条公网信令都包含 `from_device_id`、`to_device_id`、`message_id`、`timestamp`、`nonce`、`ciphertext`、`signature`。
- 客户端维护短期已处理 `message_id`，重复消息直接丢弃。

文件和文本数据仍然遵循“默认不加密”的产品策略，但公网信令必须端到端加密。若经公网中继发送文本或小文件，需提示当前中继是否开启了传输加密；本阶段先预留能力和提示字段，下一阶段再实现传输加密开关。

## 6. 中继文本/小文件数据面

中继仅在 4.4 所述条件下启用：

- 同子网直连失败
- 私有候选直连失败
- P2P 打洞失败
- 用户允许中继
- 数据类型和大小符合本阶段限制

流程：

1. 发送端向服务端发起 `relay_request`，包含目标设备、传输类型、文件名、大小、是否加密等元数据。
2. 服务端通知接收端有新中继请求。
3. 接收端确认后，服务端创建短期 `relay_session_id`。
4. 两端各自连接到该中继会话。
5. 服务端按流转发，不保存内容。
6. 传输完成、失败或超时后立即释放中继会话。

服务端配置：

- `relay_enabled`
- `relay_max_bytes`（默认 64 MB）
- `relay_session_timeout`
- `relay_max_active_sessions`

CLI 在中继传输时提示路径、上限和加密状态。超过上限时则输出：

```text
direct connection failed; relay refused: file exceeds 64MB limit
```

## 7. CLI 命令与配置

### 7.1 CLI 命令

```bash
bitexchange-cli init --root <path> --device-name <name>
bitexchange-cli listen --root <path> --port <port>
bitexchange-cli online --root <path> --server <url>
bitexchange-cli peers --root <path>
bitexchange-cli send-text --root <path> --to <device-id> --message <text>
bitexchange-cli send-file --root <path> --to <device-id> --file <path>
bitexchange-cli path-test --root <path> --to <device-id>
```

`path-test` 仅执行路径探测，不上送真实文件，便于验证当前会选哪条路径。
`send-text`/`send-file` 发送前自动调用路径选择器。

### 7.2 服务端命令

```bash
bitexchange-server \
  --listen :8080 \
  --public-url https://example.com \
  --relay-enabled=true \
  --relay-max-bytes 67108864 \
  --relay-session-timeout 5m
```

### 7.3 客户端配置

配置仍放在根目录下：

```text
bitExchange/
  chat.txt
  receivedFiles/
  device.json
  trusted-peers.json
  config.json
```

`config.json` 包含：信令服务器地址、是否允许中继、中继限制大小、监听端口以及最近成功路径缓存。CLI 每次通过 `--root` 读取配置和设备身份，不依赖全局状态。

## 8. 测试与验收标准

### 8.1 自动化测试

**服务端在线注册**：注册设备 ID/指纹/端口/候选地址；心跳超时后从在线表移除。

**候选地址交换**：在线设备 A/B 互相获取对方私有候选地址；未配对或非可信设备不能查询。

**路径选择器**：同子网可达时优先直连；同子网不可达但私有候选可达时走向私有候选直连；私有候选失败后进入 P2P 协商；P2P 失败且允许中继时走中继；禁用中继或超出中继上限时返回明确错误。

**端到端信令加密**：加密信令可被可信目标解密并验证签名；指纹不匹配、签名错误、重复 message_id、非可信设备消息均会被拒绝。

**中继文本与小文件**：文本可通过中继抵达并写入接收端 `chat.txt`；小文件可通过中继保存到接收端 `receivedFiles/`；超出 `relay_max_bytes` 的文件被拒绝且错误信息明确。

### 8.2 手工验收场景

- 同子网两客户端 → 直接局域网路径。
- 同站点不同私有子网两客户端 → 公网服务器交换私有候选地址后以私有网络直连。
- 私有/P2P 均失败 → 允许中继时文本和小文件通过中继；关闭中继或超出上限时明确失败。

### 8.3 完成标准

- `go test ./...` 全部通过。
- `path-test` 能显示最终选中的路径。
- `send-text` 和 `send-file` 至少能通过直连和中继两种路径完成。
- 服务端不保存聊天记录或文件。
- 公网信令在服务端只体现为密文载荷。
