# bitExchange — 第三阶段桌面客户端

日期：2026-06-17

## 1. 范围

第三阶段在 Phase 1/2 的 Go core 与公网互联能力之上，构建 Windows/Linux 桌面客户端，提供聊天式文件互传的图形界面。本阶段不做 Android、Web 和托盘常驻。

新增能力：

- Tauri + Web 前端桌面壳，玻璃拟态深空风视觉
- Go core 作为 sidecar 子进程，复用 Phase 1/2 全部能力
- core 本地 HTTP API，供前端调用设备/消息/文件/配置
- 设备列表、聊天/传输流、任务面板、输入区四区主界面
- 单聊文本、单发文件、群发到多设备
- 设置页：设备、网络、安全、关于四个区块

## 2. 视觉风格

玻璃拟态深空风：深紫渐变底（`#0f0c29` → `#302b63`），半透明毛玻璃面板（`rgba(255,255,255,0.06~0.10)` + `backdrop-filter: blur`），柔和边框（`rgba(255,255,255,0.1~0.2)`），强调色用紫色渐变（`#7c3aed` → `#a78bfa`）。无衬线字体。在 Windows 和 Linux 上视觉保持一致。

## 3. 总体架构

桌面端由两部分组成：Go core 子进程和 Tauri 前端。

Go core 作为 Tauri 的 sidecar 子进程随主程序启动，监听一个随机的 `127.0.0.1:<port>` 端口，把 Phase 1/2 已有的能力封装成一组本地 HTTP 接口供前端调用：设备发现、在线注册、配对、路径选择、文本/文件收发、群发任务管理、配置读写、聊天历史读取。core 复用现有 `internal/transfer`、`internal/signaling`、`internal/pathselector`、`internal/relay`、`internal/server`、`internal/history`、`internal/config`、`internal/pairing`、`internal/crypto` 包，不重写传输逻辑。

Tauri 前端用 Web 技术实现玻璃拟态深空风 UI，通过 `fetch` 调 core 的本地 HTTP 接口；文件收发等长任务用 SSE 或轮询获取进度。前端不直接做网络 IO，所有网络能力都走 core。

进程生命周期：Tauri 主进程负责拉起 core 子进程、注入端口号、监听 core 退出并在窗口关闭时优雅结束 core。

## 4. core 本地 HTTP API

core 子进程暴露一组本地 HTTP 接口（前缀 `/api/`），全部走 `127.0.0.1`，不对外。

### 4.1 设备与会话

- `GET /api/peers` — 返回当前在线可信设备列表，含设备名、指纹、连接路径、速率、状态
- `POST /api/online` — 注册到信令服务器，启动心跳
- `POST /api/offline` — 主动下线
- `POST /api/pair/start` — 发起配对，返回配对码/二维码内容
- `POST /api/pair/accept` — 接受配对并写入 trusted-peers

### 4.2 消息与文件收发

- `POST /api/send/text` — body 含 `to_device_ids[]`（支持多设备群发）、`body`、`encrypted`；返回每个目标的 `task_id` 与路径
- `POST /api/send/file` — multipart 上传，含 `to_device_ids[]`、文件、`encrypted`；返回每个目标的 `task_id`
- `GET /api/tasks` — 返回所有进行中/最近任务的状态（进度、速率、路径、错误）
- `GET /api/tasks/{id}/events` — SSE 流，推送该任务进度
- `POST /api/tasks/{id}/cancel` — 取消任务

### 4.3 接收侧

core 在本地监听一个 TCP 端口接收对端直连/中继消息，收到后写入 `chat.txt` 和 `receivedFiles/`，并通过 `GET /api/inbox/events`（SSE）把新消息/文件推给前端。

### 4.4 配置与历史

- `GET /api/config` / `PUT /api/config` — 读写 `config.json`
- `GET /api/history?limit=N` — 读取 `chat.txt` 最近 N 行
- `GET /api/device` — 返回本机设备身份与指纹

群发在 `/api/send/text` 和 `/api/send/file` 里通过 `to_device_ids[]` 实现：core 为每个目标独立建立传输任务，互不阻塞，每任务有独立 `task_id` 和路径，前端聚合显示。

## 5. 前端 UI 结构与组件

主界面三栏 + 底部输入区：

- **左栏：在线设备列表** — 每个设备显示名称、连接路径（LAN/P2P/Relay）、速率、状态；可多选实现群发
- **中栏：聊天/传输流** — 文本消息气泡（发出右靠、收到左靠），文件消息卡片，群发聚合条
- **右栏：传输任务面板** — 实时进度条、速率、路径、完成/失败状态，每任务独立
- **底部输入区** — 文本输入框、文件按钮、加密开关、发送按钮；支持拖拽/粘贴文件

组件复用：设备卡片、消息气泡、任务卡片各做一个组件，群发时复用任务卡片显示每个目标的独立进度。设置页用独立路由 `/settings`。

事件流：前端通过两个 SSE 流订阅 core —— `/api/inbox/events`（新消息/文件）和 `/api/tasks/{id}/events`（任务进度）；设备列表用 2 秒轮询 `GET /api/peers`。

## 6. core 与前端的协作流程与错误处理

### 6.1 启动流程

1. Tauri 主进程拉起 core 子进程，传入 `--root` 参数指向用户保存根目录
2. core 监听 `127.0.0.1:0`（随机端口），把实际端口打印到 stdout 第一行（如 `LISTENING 51234`）
3. Tauri 读取该端口，注入前端环境变量，前端据此构造 API base URL
4. core 启动后自动加载 `config.json`、设备身份、可信设备，并按配置决定是否连信令服务器

### 6.2 发送流程（群发文件为例）

1. 前端 `POST /api/send/file`，multipart 含文件和 `to_device_ids[]`
2. core 为每个目标创建独立 `task_id`，立即返回 `[{task_id, to_device, path}]`
3. core 对每个目标并发执行路径选择器：LAN → 私网候选 → P2P → 中继
4. 每个任务通过 `/api/tasks/{id}/events` SSE 推送进度、速率、路径切换、完成/失败
5. 任一目标失败不影响其他目标；前端在任务面板单独标红

### 6.3 接收流程

1. core 本地监听 TCP 端口，收到对端直连/中继消息
2. 文本写入 `chat.txt`，文件保存到 `receivedFiles/`（沿用 Phase 1 逻辑）
3. core 通过 `/api/inbox/events` SSE 推送 `{type:"text"|"file", from, body|filename, savedPath}` 给前端
4. 前端在聊天流里即时渲染

### 6.4 错误处理

- core 对每个 API 调用返回结构化错误：`{error:{code, message}}`，前端统一 toast 提示
- 路径选择失败且用户关闭中继：core 返回 `path_unavailable` 错误码，前端提示“无法连接到 X，请开启中继或检查网络”
- 文件超过中继上限：返回 `relay_size_exceeded`，前端提示具体上限
- core 子进程崩溃：Tauri 检测到退出码非 0，弹窗提示并提供“重启 core”按钮
- 信令断连：core 自动重连，前端设备列表显示“重连中”状态，不阻塞局域网发送

### 6.5 任务并发与取消

- core 用 `sync.Map` 管理活跃任务，每个任务一个 goroutine
- 取消通过 context 传播，立即释放连接
- 任务保留最近 100 条历史，超出按时间淘汰

## 7. 设置页与配置项

设置页用独立路由 `/settings`，玻璃拟态深空风，分四个区块：

### 7.1 设备

- 设备名称（可改，改后重新生成设备身份会丢失可信设备，需二次确认）
- 设备指纹（只读，便于在配对时人工核对）
- 默认保存根目录（文件夹选择器，修改后 `chat.txt` 和 `receivedFiles/` 跟随迁移；迁移时提示是否移动旧文件）

### 7.2 网络

- 信令服务器地址（默认 `wss://bitexchange.example.com`，可改）
- 是否允许公网中继（开关，关闭后路径选择器不选中继）
- 中继大小上限（只读显示当前服务端配置值，本阶段不可由客户端改）

### 7.3 安全

- 传输加密默认值（开关，影响发送时加密按钮初始状态；本阶段预留，实际加密开关在下一阶段实现）
- 可信设备管理（列表 + 移除按钮，移除后该设备不能再配对/收发）

### 7.4 关于

- 版本号、core 端口、日志位置（点击打开 `chat.txt` 所在目录）

### 7.5 配置落盘

- 所有设置写入根目录 `config.json`，复用 Phase 2 的 `config.ClientConfig`
- 新增字段：`device_name`、`default_save_root`、`encrypt_default`、`trusted_devices_version`
- 修改设置时 core 校验后原子写入（写临时文件再 rename），避免半写入

## 8. 测试与验收标准

### 8.1 自动化测试（Go core 侧）

- core 本地 HTTP API 单元测试：每个端点的正常路径和错误码（`path_unavailable`、`relay_size_exceeded`、配对未确认等）
- 群发任务管理器测试：多目标并发、单目标失败不影响其他、取消立即释放、任务历史淘汰
- SSE 流测试：进度推送、断线重连、多订阅者
- 配置原子写入测试：写入中断后旧文件完好

### 8.2 前端测试

- 关键组件单测：设备卡片多选、消息气泡渲染、任务进度条、文件拖拽/粘贴转任务
- 端到端冒烟（Playwright 或手工）：启动 core → 配对 → 单发文本 → 单发文件 → 群发文件 → 收到对端消息 → 收到对端文件 → 修改设置生效

### 8.3 手工验收场景

1. 两台同子网桌面端：自动发现、单聊文本、单发文件、文件落到 `receivedFiles/`
2. 跨网两台桌面端：路径选择器走 P2P 或中继，任务面板显示选中路径
3. 群发：选 3 个设备发同一文件，3 个任务独立进度，1 个失败另 2 个照常完成
4. 设置页：改默认保存路径后新文件落到新目录；关闭中继后跨网发送给出明确失败提示
5. core 崩溃恢复：kill core 后弹窗，点重启 core 后恢复在线状态

### 8.4 完成标准

- `go test ./...` 全绿
- 前端单测全绿
- 两台桌面端能完成配对、单聊、单文件、群发、接收全流程
- 玻璃拟态深空风在 Windows 和 Linux 上视觉一致
- core 子进程随窗口关闭而退出，无残留进程
