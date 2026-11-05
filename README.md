# opencode_fpk

把开源 AI 编程智能体 [OpenCode](https://github.com/anomalyco/opencode)（MIT）打包成飞牛 fnOS（飞牛 OS）可安装的 `.fpk` 应用。

当前成品：**OpenCode 2.2.5**，双架构（x86_64 + arm64），通过飞牛桌面图标在浏览器中直接使用。

---

## 一、直接安装

1. 下载 `opencode_all_2.2.5.fpk`（236 MB）— [GitHub Release 直链](https://github.com/Trojane/opencode_fpk/releases/download/v2.2.5/opencode_all_2.2.5.fpk)
2. 校验 SHA-256：`73efb29af0dd3141b0002884cadf6a8c3188a15964faac9aa87b2d0c7426d6d7`
3. 飞牛 → 应用中心 → 手动安装 → 选择该文件
4. 安装完成后在桌面点 OpenCode 图标

安装向导会提示项目目录位置。默认项目目录：

```
文件管理 → 管理员视角 → 设备全部文件 → 存储空间1 → @appdata → opencode → projects
```

首次使用需在应用内配置模型 API Key（Claude / GPT / Gemini / DeepSeek / Ollama 等），本包不内置任何密钥。

---

## 二、这个包做了什么

上游 opencode 是单文件可执行程序（Bun 打包），自带 Web UI，但它假设自己挂在网站根路径 `/`。飞牛会把它挂在 `/app/opencode` 子路径下并通过 Unix Socket 转发，所以中间需要一层网关做三件事：

```
浏览器 → 飞牛统一网关(HTTPS, 校验登录态) → app.sock
        → Go 网关（剥前缀 / 反代 / 改写 HTML·JS·CSS / 注入 shim）
        → 127.0.0.1:14096（opencode serve，官方二进制，未改一行代码）
```

1. **FPK 壳**：标准 `manifest + cmd/ + config/ + wizard/ + app/ + ICON`
2. **Go 网关**（`src/app/server/gateway.go`）：前缀剥离、反向代理、HTML/JS/CSS 路径改写、HTTP Basic Auth 自动注入、进程保活与自愈
3. **浏览器 shim**（`src/app/server/shim_gen.go`）：给 `fetch`/`XHR`/`WebSocket`/`Worker`/`EventSource` 补路径前缀，用 Service Worker 兜住动态 `import()`

服务器地址只监听 `127.0.0.1:14096` 与 Unix Socket，不做公网监听。

---

## 三、相对 2.2.1 基线修的 4 处

| # | 文件 | 改动 |
|---|---|---|
| 1 | `gateway.go` | `VERSION` 常量 → `2.2.5` |
| 2 | `gateway.go` | **修 bug**：`/gateway/status`、`/gateway/healthz` 只注册了无前缀路径，浏览器请求 `/app/opencode/gateway/status` 会被转发到上游拿到 SPA 的 HTML 再被纠正成 404，导致顶部“项目文件目录”提示条从不显示。改为两种路径都注册 |
| 3 | `shim_gen.go` | **修 bug**：一键打开项目原逻辑依赖中文文案和旧接口 `/find/file`；2.2.5 默认英文界面、API 迁到 `/api/*`，旧逻辑完全不触发。改为匹配 `Nothing here yet` / `Create a session to get started` → 点 `Add project` → 目录弹层点 `Select folder`，并限制只在 `.../projects` 路径自动确认 |
| 4 | `manifest`、`应用介绍.txt` | 版本号、changelog、文案 |

v2 的 `/api` 前缀迁移不需要改路由表：网关的 `injectShim` 有 `="/"` 通配改写，`STR_PATHS` 已含 `/api/`，运行时 shim 又会给所有同源绝对路径补前缀。

完整差异见 `patches/*.diff`，构建过程与验证记录见 `docs/构建记录-2.2.5.md`。

---

## 四、自己构建

依赖：`git` `curl` `tar` `go >= 1.21` `fnpack`

```bash
chmod +x build-latest.sh
./build-latest.sh 2.2.5          # 双架构
./build-latest.sh 2.2.5 x86      # 仅 x64
./build-latest.sh 2.2.5 arm      # 仅 arm64
```

脚本流程：拉取 2.2.1 适配层基线 → 打 `patches/` 补丁 → 下载上游官方 2.2.5 二进制 → 交叉编译网关 → `fnpack build`。

产物在 `dist/`。国内直连 GitHub 会断流，脚本默认用 `ghfast.top` 镜像（`OC_MIRROR` 可改），失败自动回退直连。

`fnpack` 下载：<https://static2.fnnas.com/fnpack/fnpack-1.2.3-linux-amd64>

---

## 五、验证情况

已在 Linux 沙箱内做端到端验证（非飞牛实机）：

- `serve` 启动后网关捕获随机密码（43 字节），Basic Auth 注入正常
- 首页经网关改写后 11,150 B，页面引用的 92 个静态资源全部 200
- Chromium 实际渲染：项目列表、设置页、模型/提供商面板正常
- `/api/event`（SSE）、`/api/info`、`/api/project`、`/api/session`、`/api/model`、`/api/provider`、`/api/integration` 全部 200
- 首次进入自动打开项目、顶部路径提示条正常

**未验证**：飞牛实机安装/升级、arm64 运行时、飞牛统一网关对 SSE 长连接的转发超时。

控制台会出现 `/app/opencode/provider`、`/app/opencode/agent` 两条 404，是 2.2.5 前端自带的旧路径探测（真实数据走 `/api/*`），官方原生部署同样会 404，不影响功能。

---

## 六、许可

- 上游 OpenCode：MIT
- 适配层（网关、shim、打包脚本）：Apache-2.0
- 本仓库为社区第三方打包，上游官方团队未参与制作；包内程序为上游官方原版二进制
