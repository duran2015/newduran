# 通用游戏机器 → Speeding 小故事 → 傻瓜式工具：执行路线

状态：待产品审核  
总原则：先造一台简单的游戏机器（架构设计时考虑未来扩展性），再用一个很小的 Speeding 故事证明它能工作；机器跑稳后，才把安装内容的过程做成傻瓜式工具。

产品验收不依赖阅读代码或接口。每个阶段的页面、固定用例、通过标准和签字方式见 `game-machine-product-acceptance-plan.md`。任何 M/S/T 版本没有对应的可点击验收场景，不视为交付。

## 1. 三条版本线

```text
M 线：Machine
  做通用游戏机器，不写 Speeding 剧情

S 线：Speeding Story
  用小故事验收机器，不新增专用底层逻辑

T 线：Tooling
  机器和 Schema 稳定后，把配置过程产品化
```

执行顺序固定为：

```text
M0 → M1 → M2 → M3 → S0 → S1 → 稳定性观察 → T0 → T1
```

任何阶段都不能为了页面演示跳过权威状态、版本、幂等、权限或复盘。

## 2. 最终形态

```text
Agent Hub
  Agent / Relation / Group / Workflow / Memory / Extension / Audit
                        │
                        ▼
Game Machine
  Package Loader / Session / Turn / Command / Rule / Effect /
  Clock / Ending / Random / Replay / Observation / Presentation Cue
                        │
             ┌──────────┴──────────┐
             ▼                     ▼
Speeding Package               未来其他 Game Package
世界/人物/事件/规则/素材         不修改 Game Machine
             │
             ▼
Player App + Presentation Runtime
自然语言 / 画面 / 音乐 / 进度 / 结局 / 复盘
```

Content Studio 最后生成标准 Game Package，不拥有第二套运行逻辑。

## 3. M 线：简单的通用游戏机器

## M0：扩展协议与骨架

目标：先定义“游戏机器吃什么、吐什么”，暂不制作正式故事。

### 任务

1. 定义 `GamePackageManifest v1`：
   - package、版本、依赖、权限、兼容的 Kernel 版本；
   - 世界 Schema、动作、效果、事件、结局和素材入口；
   - 迁移、fallback、作者和许可证元数据。
2. 定义稳定协议：
   - `GameSession`、`GameTurn`；
   - `Command`、`ActionPlan`、`ActionProposal`；
   - `DomainEvent`、`Effect`、`RandomDraw`；
   - `Observation`、`GameView`、`PresentationCue`。
3. 建立 Registry：
   - Action、Effect、Fact、Condition、Event、Projection、Ending；
   - 扩展通过 manifest 注册，不修改核心大 `switch`。
4. 确定版本与稳定 ID 规则。
5. 定义 Agent Hub Adapter 接口，不把游戏代码写进 Hub Core。
6. 准备两个无 UI 技术夹具：
   - 计数器：验证 Command → Effect → Event → Replay；
   - 餐厅：库存、采购、供应中断和停业结局，验证领域可替换。

### 可视化验收

新增“游戏机器实验室”页面，只面向开发/产品验收：

- 查看已安装 Package；
- 查看 Schema 和注册的动作/效果；
- 运行计数器和餐厅夹具；
- 查看 Command、Event、State Diff 和版本；
- 卸载餐厅包后机器仍正常。

### 退出标准

- 两个夹具不包含 Speeding 词汇；
- 新增餐厅指标、动作、事件和结局不修改 Kernel；
- 协议、Schema 和版本策略评审冻结为 v1 draft。

## M1：确定性运行内核

目标：不接模型、不接正式 Agent，也能完整开局、行动、结算、结束和复盘。

### 任务

1. Package Loader：校验、安装、启用、绑定和不可变 bundle hash。
2. Session/Turn：
   - opening；
   - presenting；
   - awaiting_player；
   - awaiting_agents；
   - resolving；
   - triggering；
   - feedback/completed。
3. Command Pipeline：权限、状态版本、幂等、验证、提交。
4. Rule Evaluator：ALL/ANY/NOT、事实目录、优先级和解释树。
5. Effect Applier：只允许 Registry 声明的状态路径和类型。
6. 事件调度：前置、权重、互斥、冷却、once 和每回合预算。
7. 固定种子 RNG 和 RandomDraw 记录。
8. PressureClock、Commitment、延迟效果和结局候选。
9. Event Log、Snapshot、状态恢复和完整重放。
10. Policy Bot：不调用 LLM 自动跑局。

### 可视化验收

在实验室页面：

- 单步推进一回合；
- 查看每条规则为何命中/未命中；
- 对比 before/after；
- 暂停、恢复、重启服务后继续；
- 输入相同 seed 重跑并比较结果；
- 模拟重复提交，确认不会重复结算。

### 退出标准

- 餐厅夹具连续运行 1,000 局无死锁；
- 相同版本、seed、Command 得到相同正式结果；
- 所有状态变化均可追溯到 Effect 和来源 Event；
- 旧快照可以恢复，日志可以重建状态。

## M2：自然语言与 Agent Hub 桥接

目标：让自然语言和多 Agent 成为机器的标准输入，但不能成为权威裁判。

### 任务

1. `PlayerIntent → ActionPlan`：
   - 目标、对象、步骤、期限、预算、保密范围；
   - 置信度、歧义、越权和资源冲突；
   - 计划预览、修改、确认和取消。
2. `ActionPlan → Command/Workflow`：
   - 直接行动；
   - 咨询；
   - 委派；
   - 会议；
   - 审批。
3. Hub Adapter：复用 H1–H7 的 Agent、关系、消息、群组、工作流、记忆和审计。
4. Observation Projector：
   - Fact、Claim、Belief、Secret 分离；
   - 每个角色只取得自己的最小观察。
5. Callback Barrier：等待、超时、fallback、取消和恢复。
6. Agent 输出限制为 Advice、Vote、Claim、ActionProposal。
7. Narrator 只根据正式 Event/State Diff 生成文本。
8. 模型不可用时的规则化降级路径。

### 可视化验收

实验室增加：

- 输入自然语言并查看结构化计划；
- 修改步骤后确认；
- 展开 Agent 咨询和消息链；
- 查看每个 Agent 收到的 Observation；
- 人为制造模型超时并查看 fallback；
- 对比 Agent 声称和系统证据。

### 退出标准

- 50 条自然语言样本覆盖单步、复合、含糊、越权和冲突指令；
- 未确认计划不改变状态；
- Agent 说“已完成”但无工具结果时不能结算；
- A→B→C 的真实交互有完整证据；
- 模型断开后测试局仍能结束。

## M3：通用表现与资源运行时

目标：机器输出通用表现信号，可自由替换美术、音乐和主题。

### 任务

1. Asset Manifest 和稳定 asset ID。
2. Presentation Cue：场景、人物、表情、文本、指标、音乐、音效和转场。
3. Domain Event → PresentationPlan Projector。
4. Web Renderer 的通用场景容器。
5. 音频状态机：平静、紧张、危机、缓解、结局。
6. 多尺寸、预加载、缓存、fallback 和低带宽模式。
7. 字幕、静音、减少动态效果和可访问性。
8. 美术/音乐主题包版本与版权来源。

### 可视化验收

- 餐厅夹具安装 A、B 两套完全不同主题；
- 不修改规则即可替换背景、角色表现、音乐和音效；
- 关闭声音/动画或资源加载失败仍能完成测试局；
- 每个 Cue 能追溯到正式 Event。

### 退出标准

- Game Core 不包含任何具体文件路径；
- 更换主题包不改变正式结算和重放结果；
- PC、移动端、静音和低带宽模式通过核心流程。

## 4. S 线：很小的 Speeding 故事

## S0：Speeding Mini Package

目标：只通过配置和标准扩展点制作故事，证明 Machine 可以承载真实产品。

### 故事范围

控制在 15–20 分钟：

- 角色：玩家、财务负责人、法务负责人、董事会对手；
- 导演为系统角色，不作为普通可聊天 NPC；
- 指标：流动性、控制权、声誉、法律风险、证据暴露、团队稳定；
- 危机钟：流动性断裂、董事会夺权；
- 事件定义总数 5 个：
  1. 资产出售付款异常；
  2. 财务调查结果；
  3. 法务披露风险；
  4. 对手利用消息发动夺权；
  5. 危机收束/结局事件；
- 结局：暂时稳住、被架空、监管冻结。

五个是事件定义总数，不是五章大型剧情。部分事件是条件后续，只需支撑一条短而完整的危机链。

### 任务

1. 编写 Speeding World Schema 和 Package Manifest。
2. 使用已有 Registry 配置动作、规则、Effect、Clock 和 Ending。
3. 在 Hub 创建角色 Agent、人格、关系和最小知识。
4. 配置每个角色的 Observation 和私有目标。
5. 编写事件文本、Agent 判断背景和正式反馈模板。
6. 制作最小资产包：
   - 1 张封面；
   - 3 个角色基础像、每人 2 个状态；
   - 3 个背景；
   - 3 张结局卡；
   - 3 层音乐；
   - 6–8 个音效。
7. 建立黄金路径、失败路径和自由策略测试。

### 硬性约束

- 不允许向 Kernel 添加 `speeding_*` 条件判断；
- 不允许 Agent 直接写指标；
- 不允许事件以固定选项限制玩家；
- 如果缺少基础能力，先判断它是否对其他游戏通用；只有通用能力才能进入新 Kernel 小版本。

### 验收

产品经理从实验室启动 Speeding Package：

- 使用自然语言完成调查、咨询、保密、谈判或披露；
- 能看到 Agent 的真实交互和不同信息；
- 忽略危机会跨回合恶化；
- 完成至少两个不同结局；
- 展开复盘，解释每个关键变化。

### 退出标准

- 故事完整跑通；
- 没有 Speeding 专用 Kernel 逻辑；
- 至少三种不同自然语言策略能形成不同后果；
- 断开模型后仍可通过 fallback 完成一局。

## S1：Speeding First Playable

目标：把已经跑通的 Package 做成普通用户可玩的独立产品页面。

### 页面

```text
/game/speeding                 开局/继续
/game/speeding/:id             游戏主界面
/game/speeding/:id/history     复盘
/game/speeding/:id/ending      结局卡
```

### 任务

- 非工程化文案和信息层级；
- 局势、人物、自然语言输入、计划确认和任务进度；
- 角色表情、背景、指标变化、危机钟、音乐和音效；
- 异步等待提示；
- PC/移动端响应式；
- 刷新恢复和重复提交保护；
- 费用、延迟和错误的友好降级；
- 线上部署、健康检查和回滚。

### 产品验收

- 陌生玩家不进入 Agent Hub，可以自行完成一局；
- 页面不出现 MCP、Schema、ToolResult 等工程词；
- 玩家始终知道系统正在等待什么；
- 15–20 分钟内能到达结局；
- 结束后能查看关键决定、人物行动和数值因果。

### 退出标准

- 5 名内部陌生玩家完成试玩；
- 无阻塞性错误；
- 关键流程 PC/移动端通过；
- 收集理解度、节奏、等待、成本和角色可信度反馈。

## 5. 稳定性观察门

S1 完成后不立即建设完整编辑器，先进行一个稳定性批次：

- 根据试玩只修 Kernel/Schema 的真实问题；
- 新增 3 个配置事件，验证无需改代码；
- 用第二个小 Package 再次验证通用性；
- 验证旧局在新 Kernel 下可恢复或只读回放；
- 冻结 `GamePackage Schema v1`、`Content Schema v1` 和 `Presentation Schema v1`。

只有三个 Schema 稳定，T 线才能开始。否则继续在配置/代码层迭代，不建设会快速过时的表单。

## 6. T 线：傻瓜式内容工具

## T0：最小 Content Studio

目标：把已经验证的手工 Package 配置过程转成页面。

### 首版页面

- 项目与版本；
- 世界指标；
- 人物与派系引用；
- 事件基本信息；
- 条件/效果表单；
- 危机钟和结局；
- 资源绑定和 Cue Sheet；
- 单事件沙盒；
- 校验、打包、发布和回滚。

### 原则

- 表单从 Schema/Registry 自动生成；
- 没有专用 renderer 时使用通用表单；
- AI 只能生成待审核草稿；
- 页面不能执行任意脚本；
- 已发布版本不可原地修改。

### 退出标准

产品经理在不改代码、不操作数据库的情况下，创建一个新事件，绑定人物、规则和素材，试玩并发布到新局。

## T1：规模化制作工具

在 T0 稳定后再增加：

- Twine 式事件关系图；
- Dialogic 式场景时间线；
- 模板与子流程；
- 多人评论、审核和冲突处理；
- 路径覆盖与批量模拟；
- 内容变更语义 diff；
- 角色/派系/地点/素材资料库；
- 创作者发布与市场能力。

## 7. 版本交付与团队拆分

每个版本只交付一个集成结果，不让产品经理逐个验收子 Agent。

| 版本 | 可并行工作包 | 集成责任 |
|---|---|---|
| M0 | Schema/Registry、Hub Adapter、实验室骨架 | 主 Agent 冻结协议 |
| M1 | Turn/Command、Rule/Effect、Replay/Test | 主 Agent 合并并跑确定性测试 |
| M2 | Intent、Observation、Hub Callback | 主 Agent 统一端到端链路 |
| M3 | Asset/Cue、Audio、Renderer | 主 Agent 验证主题可替换 |
| S0 | 内容、Agent 配置、资产包 | 主 Agent 禁止专用 Kernel 逻辑 |
| S1 | 玩家 UI、响应式、部署与试玩 | 主 Agent 统一线上验收 |
| T0/T1 | 编辑器模块、校验、沙盒、发布 | 主 Agent 保证生成标准 Package |

共享 Schema 未冻结前，不并行实现依赖它的模块。

## 8. 每版本通用门禁

每个 M/S/T 版本都必须通过：

1. 无领域硬编码；
2. Schema/API 契约测试；
3. 幂等与并发测试；
4. 固定 seed 重放；
5. 上一版本存档恢复/只读回放；
6. 服务重启和外部模型失败恢复；
7. 租户、角色可见性和扩展权限隔离；
8. Trace、日志、成本和状态解释；
9. PC/移动端核心流程；
10. 资产 fallback 与版权来源；
11. 部署迁移和回滚演练；
12. 第二 Package 不修改 Kernel 的扩展验证。

## 9. 产品审核节点

产品经理只需要在五个节点审核：

### Gate A：M0

审核机器边界、协议和实验室是否表达清楚。

### Gate B：M2

审核自然语言计划与真实 Agent 参与是否符合产品预期。

### Gate C：S0

审核小故事是否具备基本游戏性和不同后果。

### Gate D：S1

审核普通玩家是否能完整游玩。

### Gate E：T0

审核非研发人员是否真的能制作并发布内容。

## 10. 立即下一步

批准本路线后，下一版本只执行 **M0**，不提前制作五事件故事或完整玩家页面：

1. 盘点现有代码中可复用的 Run/State/Event/Extension 能力；
2. 冻结 GamePackage、Command、Event、Effect、Observation、Cue 六类核心协议；
3. 建立 Registry 和 Package Loader 骨架；
4. 做计数器与餐厅两个技术夹具；
5. 做“游戏机器实验室”页面供产品验收；
6. 提交 M0 统一可验收版本。
