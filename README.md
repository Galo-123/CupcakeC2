# 🧁 Cupcake C2 (v3.1.0)

![License](https://img.shields.io/badge/License-MIT-purple.svg)
![Version](https://img.shields.io/badge/Version-3.1.0-blue.svg)
![Build](https://img.shields.io/badge/Build-Stable-green.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)
![Rust](https://img.shields.io/badge/Rust-1.70+-orange.svg)
![Vue](https://img.shields.io/badge/Vue-3.x-42b883.svg)

**Cupcake C2** 是一款采用 **Go + Rust** 架构的高性能、隐蔽性 Command & Control 工具。专为现代红队评估和渗透测试设计，提供了极佳的视觉体验与工业级的 OpSec 加固特性。

> **"Sweet as a cupcake, sharp as a blade."**

---

## ✨ 核心特性

### 1. 现代化控制架构
- **Server**: 基于 Go (Gin + GORM) 的高并发后端，支持异步任务分发。
- **Frontend v2**: 基于 Vue.js 3 + Element Plus 构建的深色系高级 UI，按需加载优化，首屏 JS 从 **1.17MB 压缩至 12KB**。
- **Agent**: 使用 Rust 编写，具备极小的二进制体积、内存安全、且无运行时依赖。

### 2. 全方位通信协议
- **WebSocket (WS)**: 默认协议，支持 CDN 加速与域前置 (Domain Fronting)，完美模拟常规 Web 流量。
- **反向 TCP**: 提供标准的高速、稳定长连接，基于 Yamux 多路复用。
- **正向 TCP (Bind)**: 支持服务端主动连接内网 Agent，专为横向移动与穿透隔离网络设计。
- **DNS (TXT)**: 超高隐蔽性的信标模式，适用于严格的出网限制环境。

### 3. 深度 OpSec 加固
- **端到端加密**: 全链路强制使用 **AES-256-GCM** 算法 + Salt 派生密钥，确保通信内容无法被取证。
- **智能心跳行为**: 集成 **Randomized Jitter** 机制，模拟人类交互行为，规避流量统计分析。
- **Agent 无控制台输出**: 生产构建完全静默，无任何 `println!` 特征字符串残留于二进制。
- **Windows 隐身**: 启用 `windows_subsystem = "windows"`，运行时不弹出黑色 CMD 窗口。
- **反调试/反沙箱**: 集成 PEB 调试器检测、CPU/RAM/Uptime 沙箱环境检测，完全静默处置。
- **编译时加固**:
  - **Source Remapping**: 自动移除二进制文件中的本地开发路径。
  - **Self-Destruct**: 支持执行后自动销毁。
  - **UPX Support**: 可选的一键式体积压缩。

### 4. 文件传输
- **分块上传/下载**: 大文件分 2MB 块流式传输，服务端内存占用恒定，支持实时进度显示。
- **前端进度条**: 上传/下载均有实时百分比进度对话框，用户体验大幅提升。

---

## 🚀 快速开始

### 环境依赖
- **Server**: Go 1.21+
- **Frontend**: Node.js 18+
- **Agent Compiler**: Rust 1.70+ (cargo)

### 服务端部署

1. 克隆项目：
   ```bash
   git clone https://github.com/yellatiamo/CupcakeC2.git
   cd CupcakeC2/server
   ```

2. 安装前端依赖并构建：
   ```bash
   cd frontend-v2
   npm install
   npm run build
   cd ..
   ```

3. 运行服务：
   ```bash
   go run .
   ```
   *默认访问地址：`http://127.0.0.1:9999`*  
   *初始凭据：`admin` / `cupcake123`（请在系统设置中修改）*

### Agent 模板预编译

使用页面"一键生成 Payload"功能前，需先预编译 Agent 模板：

- **Windows**: 运行 `.\compile_windows.ps1`
- **Linux**: 运行 `./compile_linux.sh`

编译后的模板将自动存放至 `server/assets/`。

---

## 🛠️ 模块说明

| 路径 | 说明 |
| :--- | :--- |
| `/server` | Go 后端服务，包含 API、监听器管理与数据存储。 |
| `/server/frontend-v2` | Vue.js 3 现代化管理后台（按需加载 + 代码分割）。 |
| `/Client` | Rust Agent 源代码，支持 ws/tcp/dns/tcp_bind 特征编译。 |
| `/server/assets` | 存放用于 Patching 的预编译 Agent 模板。 |
| `/server/storage` | 本地存储：SQLite 数据库、日志与生成的 Payload（已被 .gitignore 忽略）。 |

---

## 📋 版本历史

### v3.1.0（当前版本）

**🔒 安全加固**
- CORS 策略收紧：`AllowAllOrigins=true` → 仅允许同源（127.0.0.1/localhost）
- Payload 下载接口添加路径穿越防护，替代不安全的静态目录暴露
- AdminShell WebSocket 添加帧大小限制（1MB），防止 OOM 攻击
- Agent: 所有沙箱/反调试检测完全静默化，无控制台特征字符串
- Agent: 启用 Windows 子系统，彻底消除黑色 CMD 窗口

**⚡ 性能优化**
- 前端首屏 JS 从 **1.17MB → 12KB**（Vite 代码分割 + Element Plus 按需加载）
- 服务端 `GetNextReqID` 从 Mutex 改为 `sync/atomic`（无锁，提升 3-5x）
- Dashboard 轮询间隔 3s → 15s，降低 80% 无效请求
- Agent 重连策略升级为**指数退避**（1s→2s→4s→…→60s）

**🐛 缺陷修复**
- 修复 LogsMap 无上限内存泄漏（限制为每 Agent 最多 1000 条）
- 修复 MaintenanceReset 未关闭 OutputChannel 导致的 goroutine 泄漏
- 修复前端 Dashboard 图标在 Tree-shaking 后失效（字符串引用 → 组件对象）
- 修复 Request ID 使用 `UnixNano()` 高并发下碰撞风险（改为单调计数器）
- 修复 ping 过滤使用 `strings.Contains` 误伤含 ping 字符串的正常命令
- 修复 Agent `encrypt()` 密钥长度不满足时 panic 崩溃（改为优雅降级）
- 修复前端 TerminalTabs 3 处 GBK 乱码字符

**🧹 代码清理**
- 删除客户端遗留测试文件（temp_test_2.rs, dotnet_fix.rs, test_hollowing.rs）
- 删除所有生产代码中的 `console.log` / `println!`
- 删除重复定义的 `upgrader` 变量
- 修复 `unused_variables` 编译警告

### v3.0.5
- UI 重构至 MainLayout V2，界面更专业。
- 正向 TCP (Bind) 受控端完整支持。
- 新增自动 Jitter 心跳抖动模式。
- 独立 Linux/Windows 模板构建脚本。

---

## ⚠️ 免责声明

本工具仅限于**合法的授权安全测试**。使用者需遵守当地法律法规，严禁用于非法用途。作者 Tiamo 不对任何因滥用此工具导致的损害承担责任。

---

**Developed by Tiamo | Version 3.1.0 • Build 2026**
