# 🧁 Cupcake C2 (v3.0.5)

![License](https://img.shields.io/badge/License-MIT-purple.svg)
![Version](https://img.shields.io/badge/Version-3.0.5-blue.svg)
![Build](https://img.shields.io/badge/Build-Advanced_Evasion-red.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)
![Rust](https://img.shields.io/badge/Rust-1.75+-orange.svg)
![Vue](https://img.shields.io/badge/Vue-3.x-42b883.svg)

**Cupcake C2** 是一款采用 **Go + Rust** 架构的高性能、工业级隐蔽性 Command & Control 工具。它融合了现代红队实战中的深度免杀技术（AMSI/ETW Patching）与极致的 UI 交互体验，专为突破现代 EDR/AV 监控而生。

> **"Sweet as a cupcake, invisible as a ghost."**

---

## 🛠️ 技术栈架构

Cupcake 采用了经典的高性能混合开发模式，确保了服务端的高并发处理能力与客户端的极致轻量化：

| 模块 | 核心技术 | 优势 |
| :--- | :--- | :--- |
| **Server** | **Go (Gin, GORM, SQLite)** | 高并发处理、低内存占用、易于部署。 |
| **Agent** | **Rust (Tokio, Yamux, WinAPI)** | 内存安全、无额外运行时、底层的系统调用控制能力。 |
| **Frontend** | **Vue 3 (Vite, Element Plus)** | 极致的响应速度、现代化深色系 UI、按需加载优化。 |
| **Build System** | **Cargo + CI/CD Scripts** | 支持跨平台交叉编译与自动化源代码脱敏。 |

---

## ✨ 核心免杀与 OpSec 特性 (New)

在最新的 v3.0.5 版本中，我们引入了多项深层对抗技术：

### 1. 动态自修复补丁 (Dynamic Patching)
- **AMSI Bypass**: 在运行时动态定位 `amsi.dll` 中的 `AmsiScanBuffer` 偏移，通过汇编级操作（RET 指令覆盖）屏蔽内存扫描，使本地杀软（360/火绒/Defender）对内存中的 payload 失去感知。
- **ETW Telemetry Blinding**: 针对 EDR 对系统调用的监控，直接对 `ntdll.dll` 中的 `EtwEventWrite` 进行 Patch，彻底切断 EDR（如卡巴斯基、火绒）的行为遥测链。

### 2. 流量与内存混淆 (Evasion)
- **Stealthy Memory Ballooning**: 采用**渐进式分布式内存气球**技术，通过模拟大型应用（如浏览器）的启动内存分配行为，诱导云沙箱放弃分析并规避本地杀软的瞬间大内存申请预警。
- **Sleep Obfuscation Bypass**: 弃用传统 `sleep` API，采用空转 CPU 计算质数的方式进行启动延迟。绕过 EDR 对休眠函数的 Hook。
- **流量混淆**:
  - **WebSocket + TLS**: 支持云 CDN 转发与域前置。
  - **Packet Obfuscation**: 可选 `base64` 文本伪装或 `junk` 垃圾数据填充，对抗 DPI 特征提取。

### 3. 多路复用通信 (Protocols)
- **Yamux Multiplexing**: 在单个 TCP/WS 连接内复用无限个流（Shell, FS, Socks5）。
- **Bind-TCP Mode**: 专为隔离网横向移动设计，支持服务端主动探测与定时重连重试（Backoff 机制）。
- **DNS (TXT) Tunnel**: 基于 DNS 查询的隐蔽心跳与指令传输。

### 4. 极致的前端体验
- **首屏优化**: 通过 Vite 代码分割，首屏核心资源文件从 **1.1MB 压缩至 12KB**。
- **实时进度反馈**: 文件的分块渲染与上传/下载实时进度条支持，内存占用恒定。

---

## 🚀 部署指南

### 环境准备
- **Go 1.21+**
- **Rust 1.75+** (需安装 `x86_64-pc-windows-msvc` 目标以便交叉编译)
- **Node.js 18+**

### 快速启动
1. **构建前端**: `cd server/frontend-v2 && npm install && npm run build`
2. **初始化模板**: 运行根目录下的 `compile_windows.ps1` 进行 Agent 模板预编译。
3. **启动服务端**: `cd server && go run .`
4. **访问控制台**: `http://127.0.0.1:9999` (admin / cupcake123)

---

## 📋 版本历史

### v3.0.5（当前版本）
- **[Feature]** 集成 AMSI/ETW 运行时汇编级 Patch。
- **[Feature]** 引入 Progressive Memory Ballooning 反沙箱模块。
- **[Optimization]** 优化 Bind-TCP 重连逻辑，支持指数退避退避算法。
- **[Security]** 移除生产版本中所有的 Agent fatal 错误本地日志输出，确保 0 足迹。
- **[Bugfix]** 修复了动态编译下 Host 和心跳间隔参数传递丢失的问题。
- **[UI]** 适配深色系现代化 Layout V2 布局。

---

## ⚠️ 免责声明

本工具仅限于**合法的授权安全测试**。使用者需遵守当地法律法规，严禁用于非法用途。作者不对于任何因滥用此工具导致的损害承担责任。

**Developed by Tiamo | Version 3.0.5 • Build 2026**
