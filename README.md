<div align="center">

# Newduran

**Agent 协作与自然语言游戏创作平台。**<br/>
在 Agent Hub 基础上，统一管理 Agent、协作关系、游戏规则、NPC、事件、场景、试玩与发布。

[![License](https://img.shields.io/badge/License-Apache_2.0_Source_Available-blue)](./LICENSE)
[![GitHub](https://img.shields.io/badge/GitHub-duran2015%2Fnewduran-181717?logo=github)](https://github.com/duran2015/newduran)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react)](https://react.dev)

[Quick Start](#quick-start) · [Documentation](#documentation) · [Ecosystem](#ecosystem) · [License](#license)

**[项目进展](docs/current-status.md) · [开发文档](docs/development.md) · [原项目中文说明](README.zh-CN.md)**

</div>

---

> [!NOTE]
> Newduran 是基于 Zerone Agent Hub 进行的深度二次开发。原项目许可证与附加条款继续适用，详见 [LICENSE](./LICENSE)。

## 当前重点

- Agent、人格、工具、技能、模型与知识库管理
- Agent 关系、消息、群组、流程、审计与协作留痕
- 六步游戏编辑器：概览、规则与世界状态、Agent/NPC、事件与推进、场景与视听、试玩与发布
- 自然语言玩家行动、统一规则结算、导演与 NPC 回应
- Speeding《雨夜决议》可试玩 Demo 与三条结局验证

## What is Zerone Agent Hub?

Zerone Agent Hub is the control plane for AI agents — a unified console to configure, deploy, and manage agents across your organization. It handles agent definitions, tool bindings, skill packages, model provider credentials, and session archives, then serves them via a manifest API that any client (Claude Code-style agent host, Electron app, or CLI) can consume.

**Six core domains:** Agent · Tool · Skill · Provider · Scene · Chat

<!-- TODO: add screenshot once UI is polished -->
<!-- <p align="center"><img src="docs/assets/screenshot.png" width="800" alt="Zerone Agent Hub dashboard"></p> -->

---

## Quick Start

```bash
git clone https://github.com/duran2015/newduran.git
cd newduran/quickstart
docker compose up
```

Open http://localhost:8080 — that's it.

See [docs/development.md](docs/development.md) for local dev setup (frontend + backend split, hot reload).

---

## Features

- **Agent management** — system prompts, permissions, icons, sub-agent / tool / skill bindings
- **Tool registry** — register callable tools for agents
- **Skill packages** — SKILL.md packs with OSS-backed presigned downloads
- **Provider presets** — Anthropic / OpenAI / GLM / Kimi / Bailian, AES-GCM encrypted credentials
- **Scene templates** — preset scenarios bound to agents + prompt templates
- **Session archive** — clients push chat sessions for admin audit (Markdown / tool_use / reasoning rendering)
- **CLI native** — manage everything from the terminal via `@zerone-agent/zhub` (works with Claude Code, Cursor)

---

## Architecture

Zerone Agent Hub is a Go (Gin + GORM) backend with an embedded React 19 SPA. It coordinates with [agent-runtime](https://github.com/zerone-agents/agent-runtime) (Docker-based execution engine) and uses Casdoor for SSO, MySQL for storage, and S3-compatible OSS for skill files.

For full architecture diagrams, deployment topology, and project structure, see [docs/architecture.md](docs/architecture.md).

---

## Ecosystem

| Project | Description |
|---------|-------------|
| **Zerone Agent Hub** (this repo) | Management console — the front door |
| [open-agent-sdk](https://github.com/zerone-agents/open-agent-sdk) | SDK for building agents |
| [agent-runtime](https://github.com/zerone-agents/agent-runtime) | Docker-based agent execution engine |

---

## Documentation

- 📐 [Architecture & Project Structure](docs/architecture.md)
- ⚙️ [Configuration (Environment Variables)](docs/configuration.md)
- 📡 [API Reference](docs/api-reference.md)
- 💻 [Local Development](docs/development.md)
- 🚢 [Deployment](docs/deployment.md)
- 🔐 [Security Notes](docs/security.md)
- 🗄️ [Data Model](docs/data-model.md)

---

## Contributing

We welcome contributions! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

Quick conventions:
- Conventional Commits (`feat:` / `fix:` / `refactor:` / `chore:` / `style:`)
- DB fields: `snake_case` · JSON: `camelCase` · Go: `PascalCase`
- `gofmt -l .` should output nothing before commit

---

## License

Zerone Agent Hub is **source-available** under the [Zerone Agent Hub License](./LICENSE).

**✅ Free to use:**
- Individuals and hobbyists
- Education and research
- Internal use within a single organization (including multiple workspaces)
- Building custom solutions on top of Zerone Agent Hub backend/CLI

**💰 Requires authorization:**
- Providing Zerone Agent Hub as a hosted service to third parties (SaaS, managed service)
- Removing or modifying the Zerone Agent Hub branding in the UI (white-label)

For commercial licensing, contact: **zerone-agents@proton.me**

See [LICENSE](./LICENSE) for full terms.

---

<div align="center">

© 2025-2026 Zerone · Released under the [Zerone Agent Hub License](./LICENSE)

</div>
