# CupcakeC2 v3.0.1 Wasm Skill API 参考手册

这是为 AI Agent (MCP) 提供的插件编写规范。所有插件必须通过 `extern "C"` 调用宿主提供的隐身网关。

## 1. 核心接口 (Exports)

```rust
#[no_mangle]
pub extern "C" fn run_skill(pid: u32) {
    // 插件入口逻辑
}
```

## 2. 宿主系统调用 (Host System Calls)

通过 `host_call(api_hash, &params)` 调用 Agent 内部功能。

| 功能名称 | API Hash | 参数结构体 | 说明 |
| :--- | :--- | :--- | :--- |
| **OpenProcess** | `0x21B3FD10` | `OpenProcessParams` | [Win] 打开进程句柄 |
| **ReadProcessMemory** | `0x54C6A9B2` | `ReadProcessParams` | [Win] 读取内存 |
| **TcpConnect** | `0xA1B2C3D4` | `TcpConnectParams` | [Win/Linux] TCP 连通性测试 (端口扫描) |

### 定义导入函数

```rust
extern "C" {
    fn host_call(api_hash: u32, params_ptr: *const u8) -> u64;
    fn host_report_result(data_ptr: *const u8, length: u32);
}
```

## 3. 参数结构体 (C-Packed)

```rust
#[repr(C, packed)]
struct OpenProcessParams {
    dw_desired_access: u32,
    b_inherit_handle: u32,
    dw_process_id: u32,
}

#[repr(C, packed)]
struct ReadProcessParams {
    h_process: u64,
    base_address: u64,
    buffer_ptr: u32,
    size: u32,
    bytes_read: u32,
}

// 🌐 网络扫描参数 (新)
#[repr(C, packed)]
struct TcpConnectParams {
    target_ip: u32,  // Big-endian IPv4 Address
    port: u16,
    timeout_ms: u32,
}
```

## 4. 最小化开发模板 (No-std)

```rust
#![no_std]
#![no_main]

use core::panic::PanicInfo;

extern "C" {
    fn host_report_result(data_ptr: *const u8, length: u32);
    fn host_call(api_hash: u32, params_ptr: *const u8) -> u64;
}

#[no_mangle]
pub extern "C" fn run_skill(_pid: u32) {
    let msg = "Start...\n";
    unsafe { host_report_result(msg.as_ptr(), msg.len() as u32); }
    
    // ... 你的逻辑 ...
}

#[panic_handler]
fn panic(_info: &PanicInfo) -> ! {
    loop {}
}
```
