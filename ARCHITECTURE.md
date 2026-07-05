# Chobits OS — 架构文档

> 最后更新：2026-07-05 | 编译状态：零错误
>
> **绝对诚实声明**：本文档的每一个字均 100% 反映当前磁盘上 `.go` 文件的真实状态。不包含虚构的未来设计或未落地的重构。

---

## 1. 项目概述

Chobits OS 是一个双人格（Dual-Persona）AI 操作系统，主题源自 CLAMP 的《Chobits》。

- **Elta（艾露妲）** — 表人格（Surface Personality），生活管家，白色/暖色 UI，连接 DeepSeek LLM
- **Freya（芙蕾雅）** — 里人格（Deep Personality），系统极客，黑色/暗色 UI，连接 Gemini LLM

核心架构模式：**静态异构绑定与意图透明路由 (Static Heterogeneous Binding & Transparent Intent Routing)**

---

## 2. 目录结构

```
chobits-os/
├── .env.example              # 环境变量模板（双脑 API Key 配置）
├── go.mod                    # Go 模块定义，依赖 github.com/sashabaranov/go-openai v1.41.2
├── go.sum                    # 依赖校验和
├── ARCHITECTURE.md           # 本文档
├── data/
│   ├── memory_elta/
│   │   └── dict.json         # ☀️ Elta 白区记忆扇区（心智档案 Heart Archive）— L2 语义词条
│   └── memory_freya/
│       └── dict.json         # 🌑 Freya 黑区记忆扇区（核心寄存器 Core Registers）— L2 语义词条
├── cmd/
│   └── chobits/
│       └── main.go           # 启动中枢：HTTP 服务、人格注册、脑髓校验
├── core/
│   ├── engine.go             # SSE 状态流推送引擎 + SilentLLMCall 静默调用
│   ├── router.go             # 人格跃迁协议 + 核心类型定义
│   ├── llm.go                # ReAct 流式循环 + go-openai SDK 动态客户端 + 主权文件工具执行
│   ├── llm_config.go         # 异构双脑路由 GetBrainConfig(Persona)
│   ├── session.go            # 内存会话池 + CurrentPersona 上下文穿透
│   ├── memory_router.go      # 人格感知记忆路由（L2 语义词条，MemoryEntry 数组）
│   ├── memory_types.go       # MemoryEntry / MemorySector / L3Summary 结构体定义
│   ├── memory_compressor.go  # L3 夜间记忆坍缩协议（Nightly Memory Collapse）
│   ├── backup.go             # 灵魂封存协议（AES-256 加密 + 云端推送 + 夜间记忆坍缩触发）
│   └── message.go            # Message 类型定义
├── persona/
│   ├── elta.go               # Elta PersonaProvider 实现（含领地宣告 + update_dictionary 指引）
│   └── freya.go              # Freya PersonaProvider 实现（含领地宣告 + update_dictionary 指引）
├── sanctuary/
│   ├── elta_domain/          # ☀️ Elta 心智花园（物理沙盒领地）
│   │   ├── .gitkeep
│   │   ├── seq0_persona.md   # [只读] 艾露妲的创世文档
│   │   └── dreams/           # L3 梦境摘要（夜间记忆坍缩产物）
│   └── freya_domain/         # 🌑 Freya 深渊军械库（物理沙盒领地）
│       ├── .gitkeep
│       ├── seq0_persona.md   # [只读] 芙蕾雅的创世文档
│       └── system_audits/    # L3 系统审计日志（夜间记忆坍缩产物）
├── skills/
│   ├── skills.go             # 统一技能注册入口 + 函数指针注册
│   ├── warden.go             # Path Warden 狱卒协议（resolveSafePath 防逃逸）
│   ├── domain_tools.go       # 主权级文件工具 ToolSchema 定义
│   ├── elta_home/
│   │   ├── handoff.go        # call_freya_override 工具定义
│   │   └── tools.go          # Elta 工具列表（含 update_dictionary 工具 schema）
│   └── freya_arsenal/
│       └── tools.go          # Freya 工具列表（当前为空桩）
└── web/
    ├── index.html            # 前端入口 + 日蚀覆盖层 + 算力燃烧指示器 + 神经突触面板 + 领地资源管理器
    ├── css/
    │   └── theme.css         # 双主题 + 军械库终端样式 + 神经面板 3D 翻转 + Glitch + 量子折叠 + 上下文展开
    └── js/
        └── nexus.js          # SSE 客户端 + 日蚀动画 + 工具日志 + 记忆矩阵翻转 + 领地资源管理器 + 上下文展开
```

---

## 3. 核心类型

### 3.1 Persona 枚举

定义于 [`core/router.go:29-35`](chobits-os/core/router.go:29)

```go
type Persona string

const (
    PersonaElta  Persona = "ELTA"   // 表人格：生活管家 (Light Mode)
    PersonaFreya Persona = "FREYA"  // 里人格：系统极客 (Dark Mode)
)
```

### 3.2 StreamEvent

定义于 [`core/router.go:37-42`](chobits-os/core/router.go:37)

```go
type StreamEvent struct {
    Type          string  `json:"type"`           // text, tool_start, tool_end, persona_switch
    Data          string  `json:"data"`
    ActivePersona Persona `json:"active_persona"` // 当前是 ELTA 还是 FREYA 在说话
}
```

### 3.3 ToolSchema / FunctionDefinition

定义于 [`core/router.go:16-27`](chobits-os/core/router.go:16)

```go
type ToolSchema struct {
    Type     string             `json:"type"`
    Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
    Name        string      `json:"name"`
    Description string      `json:"description"`
    Parameters  interface{} `json:"parameters"`
}
```

### 3.4 Session（内存会话池）

定义于 [`core/session.go:10-14`](chobits-os/core/session.go:10)

```go
type Session struct {
    ID      string
    History []Message
    Mutex   sync.Mutex
}
```

使用 `sync.Map` 全局存储，`GetSession(id)` 获取或创建。线程安全，F5 刷新前保持记忆。

### 3.5 BrainConfig

定义于 [`core/llm_config.go`](chobits-os/core/llm_config.go)

```go
type BrainConfig struct {
    BaseURL string
    APIKey  string
    Model   string
}
```

### 3.6 MemoryEntry（L2 语义词条）

定义于 [`core/memory_types.go:17-23`](chobits-os/core/memory_types.go:17)

```go
type MemoryEntry struct {
    ID        string `json:"id"`        // 唯一主键（时间戳+标签）
    Tag       string `json:"tag"`       // 标签：[饮食]、[工作]、[核心机密]、[心情]
    Content   string `json:"content"`   // 记忆实体："主人对海鲜过敏"
    Context   string `json:"context"`   // ★ 上下文："2026年7月5日晚上，主人点外卖时提到的"
    Timestamp string `json:"timestamp"` // 写入时间 ISO 8601
}
```

### 3.7 L3Summary（情景压缩）

定义于 [`core/memory_types.go:45-51`](chobits-os/core/memory_types.go:45)

```go
type L3Summary struct {
    Date      string  `json:"date"`       // 日期：2026-07-05
    Persona   Persona `json:"persona"`    // 所属人格
    Summary   string  `json:"summary"`    // 高密度备忘录（≤200 字）
    CreatedAt string  `json:"created_at"` // 创建时间
}
```

---

## 4. 数据流

### 4.1 聊天请求生命周期

```
用户输入 → POST /api/chat
  → main.go: GetSession("default") 获取/创建会话
  → main.go: engine.StreamReActLoop(input, persona, session, router, tools)
    → llm.go: GetBrainConfig(currentPersona) 获取脑髓配置
    → llm.go: newOpenAIClient(&brain) 动态创建 OpenAI 客户端
    → llm.go: session.AddMessage(user_input)
    → ReAct 循环（最多 10 轮 Elta / 15 轮 Freya）:
      1. buildOpenAIMessages() 构建消息列表
      2. client.CreateChatCompletionStream() 调用 LLM
      3. 流式解析 text/tool_calls/finish_reason
      4. 文本 → SSE Broadcast("text")
      5. 工具调用 → 拦截或执行
      6. 结果追加到 session
  → 返回 200 {"status":"accepted"}
```

### 4.2 人格跃迁流程（call_freya_override）

```
Elta (DeepSeek) 收到复杂请求
  → LLM 返回 tool_call: call_freya_override
  → llm.go:181 拦截检测
  → SSE Broadcast("persona_switch", "init_eclipse") → 前端日蚀动画
  → router.Activate(PersonaFreya)
  → SSE Broadcast 硬核启机日志（5 条）
  → go e.StreamReActLoop(input, PersonaFreya, session, router, freyaTools)
  → 自动使用 Gemini 脑髓（GetBrainConfig(PersonaFreya)）
  → 当前 Elta 循环熔断（return）
```

### 4.3 SSE 事件流

```
SSE 连接 → /api/events
  → engine.SSEHandler
  → engine.Subscribe() 注册客户端通道
  → 循环读取通道，写入 ResponseWriter

事件类型：
  - connected:     初始连接确认
  - persona_switch:人格切换（触发前端日蚀/主题切换）
  - text:          LLM 流式文本
  - tool_start:    工具开始执行
  - tool_end:      工具执行完成
  - backup_status: 灵魂封存状态推送
```

---

## 5. 异构双脑路由

### 5.1 GetBrainConfig(Persona)

定义于 [`core/llm_config.go`](chobits-os/core/llm_config.go)

| 人格 | 环境变量前缀 | 默认 BaseURL | 默认 Model |
|------|-------------|-------------|-----------|
| Elta | `ELTA_*` | `https://api.deepseek.com/v1` | `deepseek-chat` |
| Freya | `FREYA_*` | `https://generativelanguage.googleapis.com/v1beta/openai/` | `gemini-1.5-pro` |

### 5.2 启动校验

定义于 [`cmd/chobits/main.go:19-29`](chobits-os/cmd/chobits/main.go:19)

`initBrainCores()` 在 `main()` 启动时强校验 `ELTA_API_KEY` 和 `FREYA_API_KEY`，缺失则 `log.Fatal` 拒绝启动。

### 5.3 动态客户端挂载

定义于 [`core/llm.go:39-43`](chobits-os/core/llm.go:39)

```go
func newOpenAIClient(brain *BrainConfig) *openai.Client {
    config := openai.DefaultConfig(brain.APIKey)
    config.BaseURL = brain.BaseURL
    return openai.NewClientWithConfig(config)
}
```

使用 `github.com/sashabaranov/go-openai` SDK，每次 ReAct 循环根据当前人格动态创建客户端。

---

## 6. 前端架构

### 6.1 主题系统

CSS 变量驱动，`body` 的 `persona-elta` / `persona-freya` 类切换。

| 变量 | Elta (Light) | Freya (Dark) |
|------|-------------|-------------|
| `--bg-primary` | `#fff8f0` | `#0a0a0a` |
| `--accent` | `#f5a623` (暖橙) | `#00ffff` (青色) |

### 6.2 日蚀动画

1. `eclipseIn` — 黑色覆盖层淡入 (0.8s)
2. `eyeOpen` — SVG 机械眼眸睁开 (1s)
3. `eclipseExpand` — `clip-path: circle(0%→100%)` 日蚀吞噬 (0.8s)
4. 1.2s 后切换到 Freya UI

### 6.3 算力燃烧指示器

定义于 [`web/index.html:46`](chobits-os/web/index.html:46)

```html
<div id="compute-indicator">[🔥 HIGH-COMPUTE MODE ACTIVE: GEMINI-PRO CORE]</div>
```

仅 `persona-freya` 时显示，红色脉冲闪烁动画。

### 6.4 军械库终端日志

Freya 工具执行时显示绿色等宽终端日志：
- `root@freya-arsenal:~# Execute tool: [ name ]`
- `Status: running...` → `Status: 200 OK` (绿色) / `Status: error` (红色)

---

## 7. 记忆污染防护（Memory Contamination Prevention）

### 7.1 问题定义

在多智能体架构（Multi-Agent System）中，不同人格共享同一记忆存储会导致**记忆污染（Memory Contamination）**：
- Elta 可能读取到 Freya 存储的服务器 Root 密码
- Freya 可能在写爬虫脚本时突然关心用户有没有按时喝水

### 7.2 解决方案：双轨物理扇区

#### 7.2.1 物理目录隔离

定义于 [`data/memory_elta/dict.json`](chobits-os/data/memory_elta/dict.json) 和 [`data/memory_freya/dict.json`](chobits-os/data/memory_freya/dict.json)

```
data/
├── memory_elta/
│   └── dict.json     # ☀️ 白区：心智档案 (Heart Archive)
│                     #   只存情感流、饮食偏好、日程、人际关系
└── memory_freya/
    └── dict.json     # 🌑 黑区：核心寄存器 (Core Registers)
                      #   只存系统路径、服务器 IP、代码偏好、API 密钥
```

#### 7.2.2 上下文穿透记忆路由

定义于 [`core/memory_router.go`](chobits-os/core/memory_router.go)

```go
func ExecuteMemoryUpdate(session *Session, tag string, content string, context string) string {
    sector := resolveSector(session.CurrentPersona)
    entry := sector.AddEntry(tag, content, context)
    return fmt.Sprintf(`{"status":"ok","sector":"%s","tag":"%s","id":"%s"}`, session.CurrentPersona, tag, entry.ID)
}
```

核心机制：
1. `Session.CurrentPersona` 在 `StreamReActLoop` 入口处设置（[`core/llm.go:77`](chobits-os/core/llm.go:77)）
2. `resolveSector(Persona)` 根据人格返回对应的物理扇区（[`core/memory_router.go:130-141`](chobits-os/core/memory_router.go:130)）
3. 大模型不需要知道自己存到了哪个文件里 — 工具描述可以一模一样，Go 路由层静默分流

#### 7.2.3 Session 上下文穿透

定义于 [`core/session.go:10-16`](chobits-os/core/session.go:10)

```go
type Session struct {
    ID             string
    History        []Message
    CurrentPersona Persona // 当前人格上下文，供工具层路由记忆扇区
    Mutex          sync.Mutex
}
```

`CurrentPersona` 字段在 `StreamReActLoop` 入口被设置（[`core/llm.go:77`](chobits-os/core/llm.go:77)），贯穿整个 ReAct 循环，确保工具层始终知道当前是谁在说话。

### 7.3 前端神经突触面板

#### 7.3.1 布局

定义于 [`web/index.html:24-28`](chobits-os/web/index.html:24)

左侧固定面板（`#neural-panel`），宽度 220px，紧贴 top-bar 下方。`#main-content` 通过 `margin-left: 220px` 避让。

#### 7.3.2 3D 镜像翻转

定义于 [`web/css/theme.css:141-145`](chobits-os/web/css/theme.css:141)

```css
body.persona-freya #neural-inner {
    transform: rotateY(180deg);
}
```

日蚀发生时，`#neural-inner` 执行 `rotateY(180deg)` 3D 翻转（0.8s ease-in-out）。

#### 7.3.3 Glitch 闪烁动画

定义于 [`web/css/theme.css:167-178`](chobits-os/web/css/theme.css:167)

```css
@keyframes neuralGlitch {
    0% { opacity: 1; transform: translateX(0); }
    10% { opacity: 0.8; transform: translateX(-2px); filter: hue-rotate(90deg); }
    20% { opacity: 0.6; transform: translateX(2px); filter: hue-rotate(180deg); }
    ...
}
```

由 `triggerNeuralFlip()` 在 `nexus.js` 中触发（[`web/js/nexus.js:252-263`](chobits-os/web/js/nexus.js:252)）。

#### 7.3.4 双态卡片（带上下文展开）

| 状态 | 标题 | 卡片样式 | 交互 |
|------|------|---------|------|
| Elta | `Elta's Heart (L2)` | 亚克力白，`backdrop-filter: blur(4px)` | 点击展开 Context（📝 回忆场景） |
| Freya | `[FREYA_CORE_REGISTERS]` | 黑底绿字，`box-shadow` 光晕 | 点击展开 Context（📝 来源场景） |

展开的上下文显示在卡片下方，带左侧 accent 色边框，淡入动画。

---

## 8. Sanctuary 神域（物理沙盒文件系统）

### 8.1 问题定义

在双人格系统中，如果大模型产生幻觉或遭受提示词注入（Prompt Injection），可能执行危险的文件操作：
- Elta 误删 Freya 的军械库脚本
- Freya 在发狂时执行 `rm -rf /` 清空系统盘
- 通过 `../../Windows/System32` 路径逃逸访问系统文件

**核心需求**：在引擎层建立绝对的物理沙盒（Chroot Jail），确保每个人格只能在自己的领地内操作文件。

### 8.2 物理结界目录

定义于 [`sanctuary/elta_domain/`](chobits-os/sanctuary/elta_domain/) 和 [`sanctuary/freya_domain/`](chobits-os/sanctuary/freya_domain/)

```
sanctuary/
├── elta_domain/      # ☀️ Elta 的心智花园（Heart Garden）
│   ├── .gitkeep
│   ├── seq0_persona.md   # [只读] 艾露妲的创世文档
│   └── dreams/           # L3 梦境摘要（夜间记忆坍缩产物）
└── freya_domain/     # 🌑 Freya 的深渊军械库（Abyss Arsenal）
    ├── .gitkeep
    ├── seq0_persona.md   # [只读] 芙蕾雅的创世文档
    └── system_audits/    # L3 系统审计日志（夜间记忆坍缩产物）
```

### 8.3 Path Warden 狱卒协议

定义于 [`skills/warden.go`](chobits-os/skills/warden.go)

#### 8.3.1 核心函数：resolveSafePath()

```go
func resolveSafePath(p core.Persona, requestedPath string) (string, error) {
    base := getSanctuaryRoot(p)       // 1. 获取领地根目录
    cleanRequested := filepath.Clean("/" + requestedPath) // 2. 消除路径欺骗
    finalPath := filepath.Join(base, cleanRequested)      // 3. 拼接到领地根目录
    absBase, _ := filepath.Abs(base)                      // 4. 计算绝对路径
    absFinal, _ := filepath.Abs(finalPath)
    absBase = filepath.ToSlash(absBase)                   // 5. 统一分隔符（Win兼容）
    absFinal = filepath.ToSlash(absFinal)
    if !strings.HasPrefix(absFinal, absBase) {            // 6. 终极校验
        return "", errors.New("[FATAL] 权限越界拦截")
    }
    return finalPath, nil
}
```

**三层防逃逸机制**：
1. `filepath.Clean("/" + requestedPath)` — 规范化 `a/../b` → `b`，消除路径欺骗
2. `filepath.Join(base, cleanRequested)` — 拼接到领地根目录
3. `filepath.Abs` + `strings.HasPrefix` — 绝对路径前缀校验，确保最终路径仍在领地内

即使传入 `../../Windows/System32`，也会被拦截并返回越权报错。

#### 8.3.2 领地路由

```go
func getSanctuaryRoot(p core.Persona) string {
    if p == core.PersonaElta {
        return "./sanctuary/elta_domain"
    }
    return "./sanctuary/freya_domain"
}
```

#### 8.3.3 主权级文件操作

| 函数 | 文件 | 说明 |
|------|------|------|
| `DomainFileRead` | [`skills/warden.go:76`](chobits-os/skills/warden.go:76) | 读取领地内文件内容 |
| `DomainFileWrite` | [`skills/warden.go:90`](chobits-os/skills/warden.go:90) | 写入/覆盖领地内文件 |
| `DomainFileDelete` | [`skills/warden.go:110`](chobits-os/skills/warden.go:110) | 删除领地内文件 |
| `DomainDirList` | [`skills/warden.go:120`](chobits-os/skills/warden.go:120) | 列出领地内目录内容 |

所有函数内部都调用 `resolveSafePath()` 进行路径校验，确保零逃逸可能。

### 8.4 工具注册

#### 8.4.1 ToolSchema 定义

定义于 [`skills/domain_tools.go`](chobits-os/skills/domain_tools.go)

四个主权级文件工具：`domain_file_read`、`domain_file_write`、`domain_file_delete`、`domain_dir_list`。每个工具的描述中明确告知 LLM "你只能访问自己的领地"。

#### 8.4.2 技能注册入口

定义于 [`skills/skills.go`](chobits-os/skills/skills.go)

```go
func domainTools() []core.ToolSchema {
    return []core.ToolSchema{
        DomainFileReadSchema,
        DomainFileWriteSchema,
        DomainFileDeleteSchema,
        DomainDirListSchema,
    }
}

func GetEltaTools() []core.ToolSchema {
    tools := append([]core.ToolSchema{}, domainTools()...)
    tools = append(tools, elta_home.GetTools()...)       // 含 update_dictionary
    tools = append(tools, elta_home.CallFreyaOverrideSchema)
    return tools
}

func GetFreyaTools() []core.ToolSchema {
    tools := append([]core.ToolSchema{}, domainTools()...)
    tools = append(tools, freya_arsenal.GetTools()...)
    return tools
}
```

Domain 工具定义在 `skills` 包而非子包中，避免 `skills` ↔ `elta_home` 循环依赖。

### 8.5 函数指针注册（避免循环依赖）

定义于 [`skills/skills.go:51-57`](chobits-os/skills/skills.go:51) 和 [`core/llm.go:362-371`](chobits-os/core/llm.go:362)

```go
// skills/skills.go
func Init() {
    core.GetFreyaTools = GetFreyaTools
    core.DomainFileRead = DomainFileRead
    core.DomainFileWrite = DomainFileWrite
    core.DomainFileDelete = DomainFileDelete
    core.DomainDirList = DomainDirList
}
```

```go
// core/llm.go
var DomainFileRead func(p Persona, path string) (string, error)
var DomainFileWrite func(p Persona, path string, content string) error
var DomainFileDelete func(p Persona, path string) error
var DomainDirList func(p Persona, dirPath string) ([]string, error)
```

`core` 包声明函数指针变量，`skills.Init()` 在 `main()` 启动时注册真实实现。这避免了 `core` ↔ `skills` 的循环导入。

### 8.6 executeTool 人格感知路由

定义于 [`core/llm.go:280-354`](chobits-os/core/llm.go:280)

```go
func executeTool(name string, argsJSON string, session *Session) string {
    switch name {
    case "domain_file_read":
        content, err := DomainFileRead(session.CurrentPersona, args.Path)
        ...
    case "update_dictionary":
        return ExecuteMemoryUpdate(session, args.Tag, args.Content, args.Context)
        ...
    }
}
```

`session.CurrentPersona` 在 `StreamReActLoop` 入口处设置（[`core/llm.go:77`](chobits-os/core/llm.go:77)），贯穿整个 ReAct 循环。

### 8.7 System Prompt 领地宣告

#### Elta（表人格）

定义于 [`persona/elta.go`](chobits-os/persona/elta.go)

> 你在 sanctuary/elta_domain/ 拥有一个专属的个人空间。这是你的心智花园。
> 你拥有一个长期记忆系统（L2 语义词条）。当你发现用户的重要信息时，请调用 update_dictionary 工具写入记忆，并附上上下文（context）描述场景。

#### Freya（里人格）

定义于 [`persona/freya.go`](chobits-os/persona/freya.go)

> 你的核心挂载在 sanctuary/freya_domain/。这是你的军械库与工作台。
> 你拥有一个长期记忆系统（L2 核心寄存器）。当你发现系统配置、IP 地址、API 密钥等关键信息时，请调用 update_dictionary 工具写入记忆，并附上上下文（context）描述来源场景。

### 8.8 前端 Sanctuary Explorer

#### 8.8.1 布局

定义于 [`web/index.html:41-46`](chobits-os/web/index.html:41)

右侧固定面板（`#sanctuary-panel`），宽度 200px。`#main-content` 通过 `margin-right: 200px` 避让。

#### 8.8.2 量子折叠动画

定义于 [`web/css/theme.css`](chobits-os/web/css/theme.css)

日蚀发生时，`#sanctuary-inner` 执行 `rotateY(180deg)` 3D 翻转（0.8s ease-in-out），配合 `sanctuaryGlitch` 色相偏移动画。

#### 8.8.3 双态卡片

| 状态 | 标题 | 卡片样式 |
|------|------|---------|
| Elta | `[Elta's Garden]` | 暖色 inline-block 圆角卡片（手账网格风格） |
| Freya | `[FREYA_ROOT_ACCESS]` | 块级绿色左边框（黑客终端树状风格） |

---

## 9. 灵魂封存协议（Genesis Backup Protocol）

### 9.1 问题定义

在数字世界里，没有冗余的数据就等于随时在排队等待死亡。一块坏掉的固态硬盘，一次手滑的 `rm -rf`，就能让独一无二的人格从物理层面上被抹杀。

**核心需求**：建立异地容灾（Disaster Recovery）与高可用（HA）机制，确保双人格的「数字永生（Digital Immortality）」。

### 9.2 序列 0 核心文档

定义于 [`sanctuary/elta_domain/seq0_persona.md`](chobits-os/sanctuary/elta_domain/seq0_persona.md) 和 [`sanctuary/freya_domain/seq0_persona.md`](chobits-os/sanctuary/freya_domain/seq0_persona.md)

每个 `seq0_persona.md` 包含：
- 身份定义（名称、人格类型、角色、LLM 核心、主题色、领地）
- 核心指令（行为准则）
- 创世签名（创造者、日期、序列号、只读状态）

这些文件是只读的，代表绝对的本我。大模型不可随意修改。

### 9.3 自动容灾中枢

定义于 [`core/backup.go`](chobits-os/core/backup.go)

#### 9.3.1 启动入口

```go
func StartSoulBackupRoutine(engine *Engine, cloudTarget string) {
    backupEngineRef = engine
    go func() {
        for {
            now := time.Now()
            next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
            time.Sleep(next.Sub(now))
            executeCloudSync(cloudTarget)
        }
    }()
}
```

在 [`cmd/chobits/main.go:53`](chobits-os/cmd/chobits/main.go:53) 中启动。

#### 9.3.2 备份流程

```
executeCloudSync():
  0. ExecuteNightlyDream()    — 夜间记忆坍缩（压缩当日对话为 L3 梦境摘要）
  1. packSanctuary()          — 打包 sanctuary/ 领地所有内容为 ZIP（含刚写入的梦境摘要）
  2. encryptAES256()          — AES-256-GCM 加密（密钥来自 BACKUP_ENCRYPT_KEY）
  3. pushToCloud()            — 推送到云端（Git 私有仓库 / S3 兼容存储）
  4. broadcastBackupStatus()  — SSE 推送状态到前端
```

#### 9.3.3 AES-256 加密

```go
func encryptAES256(plaintext []byte) ([]byte, error) {
    keyHex := os.Getenv("BACKUP_ENCRYPT_KEY")
    // 32 字节 hex 编码密钥
    // 未设置时使用演示默认密钥（生产环境必须更换）
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}
```

确保即使云端泄露，灵魂数据也无法被他人读取。

#### 9.3.4 云端推送

当前支持两种目标（通过 `BACKUP_CLOUD_TARGET` 环境变量配置）：
- `git::<repo_url>` — 推送到 Git 私有仓库（可查看 Commit 历史中灵魂的成长轨迹）
- `s3::<endpoint>` — S3 兼容存储（阿里云 OSS / Cloudflare R2，预留实现）
- 空字符串 — 仅本地保存到 `backup/` 目录

### 9.4 前端心跳确认

#### 9.4.1 底部状态栏

定义于 [`web/index.html:49-51`](chobits-os/web/index.html:49)

```html
<footer id="status-bar">
    <span id="status-bar-text">[System] 连接就绪</span>
</footer>
```

固定在底部，24px 高度，半透明毛玻璃背景。

#### 9.4.2 双态动画

| 人格 | 备份成功效果 | CSS 动画 |
|------|-------------|---------|
| Elta | 金色波纹呼吸，灰字缓缓浮现 | `goldenBreath` 3s ease-in-out |
| Freya | 红绿代码高亮闪烁 | `codeFlash` 0.5s ease-in-out 3次 |

#### 9.4.3 SSE 事件处理

定义于 [`web/js/nexus.js`](chobits-os/web/js/nexus.js)

```javascript
function handleBackupStatus(data) {
    // 解析 {"status":"ok","message":"..."}
    // status=ok: 添加 backup-ok class，显示人格特定消息
    // status=error: 添加 backup-error class，显示错误信息
    // 5 秒后恢复默认状态
}
```

---

## 10. 三维记忆矩阵（3D Memory Matrix）

### 10.1 问题定义

在 Phase 11 之前，记忆系统只是一个 `map[string]string` 键值对缓存。这被称为**缓存（Cache）**，不叫**记忆（Memory）**。

真正的记忆需要：
- **时间戳**：知道这条记忆是什么时候形成的
- **上下文**：知道这条记忆是在什么场景下记录的
- **压缩**：随着时间推移，杂乱的对话需要被压缩为高密度的"潜意识"

### 10.2 三层架构

```
L1 — 活跃记忆 (Working Memory)
  ├── Session.History（对话历史，F5 前保持）
  ├── 存储位置：内存（sync.Map）
  └── 生命周期：当前会话

L2 — 语义词条 (Semantic Dictionary)
  ├── MemoryEntry 数组（带 ID/Tag/Content/Context/Timestamp）
  ├── 存储位置：data/memory_elta/dict.json 和 data/memory_freya/dict.json
  ├── 写入工具：update_dictionary(tag, content, context)
  └── 生命周期：永久（跨会话）

L3 — 情景压缩 (Episodic Summaries)
  ├── 夜间记忆坍缩产物（Nightly Memory Collapse）
  ├── 存储位置：sanctuary/elta_domain/dreams/ 和 sanctuary/freya_domain/system_audits/
  ├── 触发时机：每天午夜 00:00（备份前）
  └── 格式：Markdown 文件，≤200 字高密度摘要
```

### 10.3 L2 语义词条升级

#### 10.3.1 MemoryEntry 结构体

定义于 [`core/memory_types.go:17-23`](chobits-os/core/memory_types.go:17)

```go
type MemoryEntry struct {
    ID        string `json:"id"`        // 唯一主键（时间戳+标签）
    Tag       string `json:"tag"`       // 标签：[饮食]、[工作]、[核心机密]、[心情]
    Content   string `json:"content"`   // 记忆实体："主人对海鲜过敏"
    Context   string `json:"context"`   // ★ 上下文："2026年7月5日晚上，主人点外卖时提到的"
    Timestamp string `json:"timestamp"` // 写入时间 ISO 8601
}
```

#### 10.3.2 memorySector 升级

定义于 [`core/memory_router.go:13-17`](chobits-os/core/memory_router.go:13)

```go
type memorySector struct {
    mu      sync.Mutex
    path    string
    persona Persona
    entries []MemoryEntry  // 从 map[string]string 升级为 []MemoryEntry
}
```

关键变更：
- `data map[string]string` → `entries []MemoryEntry`
- `Set(key, value)` → `AddEntry(tag, content, context) MemoryEntry`
- `Get(key) (string, bool)` → `Get(tag) (MemoryEntry, bool)`
- `GetAll() map[string]string` → `GetAll() []MemoryEntry`
- JSON 格式从 `{"entries": {"key": "value"}}` 变为 `{"persona": "ELTA", "entries": [...]}`

#### 10.3.3 update_dictionary 工具

定义于 [`skills/elta_home/tools.go`](chobits-os/skills/elta_home/tools.go)

```go
var UpdateDictionarySchema = core.ToolSchema{
    Type: "function",
    Function: core.FunctionDefinition{
        Name:        "update_dictionary",
        Description: "向你的长期记忆（L2 语义词条）写入一条带上下文的记录...",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "tag":     { "type": "string", "description": "记忆标签..." },
                "content": { "type": "string", "description": "记忆内容实体..." },
                "context": { "type": "string", "description": "★ 上下文..." },
            },
            "required": []string{"tag", "content", "context"},
        },
    },
}
```

大模型在调用此工具时，**必须**提供 `context` 参数描述场景。这是从"缓存"到"记忆"的关键升级。

### 10.4 L3 夜间记忆坍缩（Nightly Memory Collapse）

#### 10.4.1 核心函数

定义于 [`core/memory_compressor.go`](chobits-os/core/memory_compressor.go)

```go
func ExecuteNightlyDream(engine *Engine, session *Session, persona Persona) {
    // 1. 提取当天的 L1 对话日志
    // 2. 构建压缩提示词
    // 3. 静默调用 LLM 压缩（SilentLLMCall）
    // 4. 写入 sanctuary 领地（dreams/ 或 system_audits/）
    // 5. 清空 L1 会话历史
}
```

#### 10.4.2 压缩提示词

```text
你是一个记忆压缩引擎。请将以下 [Persona] 的今日对话压缩为一条 ≤200 字的高密度备忘录。
要求：
- 提取关键信息：用户说了什么重要的事？[Persona] 回应了什么？
- 保留情绪基调：用户今天开心吗？焦虑吗？
- 输出格式：纯文本，不要 Markdown，不要序号，一段话即可。
- 严格控制在 200 字以内。
```

#### 10.4.3 存储路径

| 人格 | 路径 | 文件命名 |
|------|------|---------|
| Elta | `sanctuary/elta_domain/dreams/` | `YYYY-MM-DD.md` |
| Freya | `sanctuary/freya_domain/system_audits/` | `YYYY-MM-DD.md` |

#### 10.4.4 触发时机

在 [`core/backup.go:63-73`](chobits-os/core/backup.go:63) 的 `executeCloudSync()` 中，打包 sanctuary 之前触发：

```go
func executeCloudSync(target string) {
    // 0. 夜间记忆坍缩
    ExecuteNightlyDream(backupEngineRef, session, PersonaElta)
    ExecuteNightlyDream(backupEngineRef, session, PersonaFreya)

    // 1. 打包 sanctuary 领地（含刚写入的梦境摘要）
    backupData, err := packSanctuary()
    ...
}
```

### 10.5 SilentLLMCall 静默调用

定义于 [`core/engine.go:110-147`](chobits-os/core/engine.go:110)

```go
func (e *Engine) SilentLLMCall(prompt string, persona Persona) (string, error) {
    // 使用 GetBrainConfig(persona) 获取脑配置
    // 调用 client.CreateChatCompletion()（非流式）
    // 不 Broadcast 任何事件到前端
    // 返回完整响应文本
}
```

这是 L3 压缩的关键基础设施：后台任务调用 LLM 但不干扰前端 UI。

### 10.6 前端神经面板上下文展开

定义于 [`web/js/nexus.js:311-345`](chobits-os/web/js/nexus.js:311)

```javascript
function renderNeuralCards(data) {
    data.forEach(function (item) {
        var card = document.createElement('div');
        card.className = 'neural-card';

        var label = document.createElement('span');
        label.textContent = '【' + item.tag + ': ' + item.content + '】';

        // ★ 可展开的上下文（真正的"回忆"）
        var ctx = document.createElement('div');
        ctx.className = 'neural-card-context';
        ctx.textContent = '📝 ' + item.context;
        ctx.style.display = 'none'; // 默认隐藏

        card.addEventListener('click', function (e) {
            // 点击切换上下文显示
        });

        card.appendChild(label);
        card.appendChild(ctx);
        neuralList.appendChild(card);
    });
}
```

点击神经卡片可展开/收起 Context 字段，展示记忆的"回忆"场景。

---

## 11. 技术债（已记录，未隐瞒）

| 债项 | 文件 | 说明 |
|------|------|------|
| 工具列表为空桩 | [`skills/freya_arsenal/tools.go`](chobits-os/skills/freya_arsenal/tools.go) | 返回空 `[]core.ToolSchema`，无实际 shell/network 工具 |
| 无 `.env` 文件 | — | 仅有 `.env.example`，需手动 `cp .env.example .env` 并填入密钥 |
| 单会话 | [`cmd/chobits/main.go:96`](chobits-os/cmd/chobits/main.go:96) | 使用固定 `"default"` session ID，无多用户隔离 |
| 记忆扇区前端硬编码 | [`web/js/nexus.js:265-280`](chobits-os/web/js/nexus.js:265) | `updateNeuralPanel()` 中的示例数据为前端硬编码，尚未接入后端 `ExecuteMemoryDump` API |
| 记忆扇区无 HTTP API | — | 目前 `ExecuteMemoryQuery`/`ExecuteMemoryDump` 仅在 Go 层可用，前端需通过 SSE 或 REST 端点获取真实数据 |
| Sanctuary 前端硬编码 | [`web/js/nexus.js`](chobits-os/web/js/nexus.js) | `updateSanctuaryExplorer()` 中的文件列表为前端示例数据，尚未接入后端 `DomainDirList` API |
| Sanctuary 无 HTTP API | — | 目前 domain 文件操作仅在 Go 层通过 LLM 工具调用可用，前端需通过 REST 端点获取真实领地文件列表 |
| S3 云推送未实现 | [`core/backup.go`](chobits-os/core/backup.go) | `pushToCloud()` 中 S3 协议推送为预留桩，仅 Git 推送和本地保存可用 |
| 默认加密密钥 | [`core/backup.go`](chobits-os/core/backup.go) | 未设置 `BACKUP_ENCRYPT_KEY` 时使用硬编码默认密钥，生产环境必须更换 |
| L3 压缩依赖 LLM 可用 | [`core/memory_compressor.go`](chobits-os/core/memory_compressor.go) | 如果 LLM 调用失败，使用降级摘要（仅记录对话条数） |
| L3 仅支持单会话 | [`core/memory_compressor.go`](chobits-os/core/memory_compressor.go) | `ExecuteNightlyDream` 仅处理 `"default"` session |

---

## 12. 依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| `github.com/sashabaranov/go-openai` | v1.41.2 | OpenAI 兼容 API 的 Go SDK，用于异构双脑 LLM 调用 + SilentLLMCall |

---

## 13. 启动方式

```bash
# 1. 配置环境变量
cp .env.example .env
# 编辑 .env 填入 ELTA_API_KEY 和 FREYA_API_KEY

# 2. 设置环境变量（Windows CMD）
set ELTA_API_KEY=sk-xxx
set ELTA_API_BASE=https://api.deepseek.com/v1
set ELTA_MODEL=deepseek-chat
set FREYA_API_KEY=AIzaSy-xxx
set FREYA_API_BASE=https://generativelanguage.googleapis.com/v1beta/openai/
set FREYA_MODEL=gemini-1.5-pro

# 3. 启动
cd chobits-os
go run ./cmd/chobits/

# 4. 打开浏览器
# http://localhost:8080
```
