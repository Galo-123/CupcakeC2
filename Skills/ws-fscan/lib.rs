#![no_std]
#![no_main]

use core::panic::PanicInfo;
use core::str::FromStr;

extern "C" {
    fn host_call(api_hash: u32, params_ptr: *const u8) -> u64;
    fn host_report_result(data_ptr: *const u8, length: u32);
}

// Host Call Hashes
const HASH_TCP_CONNECT: u32 = 0xA1B2C3D4;
const HASH_GET_ARGS: u32 = 0x11223344;
const HASH_TCP_SEND_RECV: u32 = 0x55667788;
const HASH_GET_NET_INFO: u32 = 0xC0FFEE10;

#[repr(C, packed)]
struct TcpConnectParams { target_ip: u32, port: u16, timeout_ms: u32 }

#[repr(C, packed)]
struct TcpSendRecvParams { 
    target_ip: u32, 
    port: u16, 
    timeout_ms: u32, 
    send_ptr: u32, 
    send_len: u32, 
    recv_ptr: u32, 
    recv_len: u32 
}

#[repr(C, packed)]
struct GenericBufferParams { buffer_ptr: u32, buffer_size: u32 }

const PORT_HTTP: u16 = 80;
const PORT_HTTPS: u16 = 443;
const PORT_SMB: u16 = 445;
const PORT_MYSQL: u16 = 3306;
const PORT_REDIS: u16 = 6379;
const PORT_WEBLOGIC: u16 = 7001;
const PORT_SPRING: u16 = 8080;
const PORT_ELASTIC: u16 = 9200;

#[no_mangle]
pub extern "C" fn run_skill(_pid: u32) {
    report("\n[+] ==========================================\n");
    report("[+]        CupcakeC2 FScan Full v3.2        \n");
    report("[+] ==========================================\n\n");

    let mut args_buf = [0u8; 256];
    let args_params = GenericBufferParams { buffer_ptr: args_buf.as_ptr() as u32, buffer_size: args_buf.len() as u32 };
    let args_len = unsafe { host_call(HASH_GET_ARGS, &args_params as *const _ as *const u8) };
    
    let mut target_ip_base = 0u32;
    let mut scan_mask = 24u8;

    if args_len > 0 {
        let args_str = unsafe { core::str::from_utf8_unchecked(&args_buf[..args_len as usize]) };
        if let Some(target) = extract_json_val(args_str, "target") {
            if let Some((ip, mask)) = parse_cidr(target) { target_ip_base = ip; scan_mask = mask; }
        }
    }

    if target_ip_base == 0 {
        let mut net_buf = [0u8; 64];
        let net_params = GenericBufferParams { buffer_ptr: net_buf.as_ptr() as u32, buffer_size: net_buf.len() as u32 };
        let net_len = unsafe { host_call(HASH_GET_NET_INFO, &net_params as *const _ as *const u8) };
        if net_len > 0 {
            let net_str = unsafe { core::str::from_utf8_unchecked(&net_buf[..net_len as usize]) };
            if let Some((ip, _)) = parse_cidr(net_str) {
                target_ip_base = ip & 0xFFFFFF00;
                report("[*] Use Local Segment: "); report_ip(target_ip_base); report("/24\n");
            }
        }
    }

    if target_ip_base == 0 {
        report("[!] Usage: set args {\"target\": \"192.168.1.0/24\"}\n");
        return;
    }

    let num_hosts = 1u64 << (32 - scan_mask);
    let start_ip = target_ip_base & (u32::MAX << (32 - scan_mask));
    let cap = if num_hosts > 256 { 256 } else { num_hosts as u32 };

    let ports = [22, 135, PORT_SMB, PORT_HTTP, PORT_HTTPS, 1433, 1521, PORT_MYSQL, 3389, PORT_REDIS, PORT_WEBLOGIC, PORT_SPRING, PORT_ELASTIC];

    for i in 0..cap {
        let current_ip = start_ip | i;
        if i == 0 || i == 255 { continue; }
        for &port in ports.iter() {
            if check_port(current_ip, port, 120) {
                report_open(current_ip, port);
                dispatch_poc(current_ip, port);
            }
        }
    }
}

fn dispatch_poc(ip: u32, port: u16) {
    match port {
        PORT_HTTP | PORT_HTTPS | PORT_SPRING => poc_web_all(ip, port),
        PORT_WEBLOGIC => { poc_web_all(ip, port); poc_weblogic_t3(ip, port); },
        PORT_SMB => poc_ms17_010(ip, port),
        PORT_REDIS => poc_redis_unauth(ip, port),
        PORT_MYSQL => poc_mysql_banner(ip, port),
        PORT_ELASTIC => poc_elasticsearch_unauth(ip, port),
        _ => {}
    }
}

fn poc_web_all(ip: u32, port: u16) {
    let req = b"GET / HTTP/1.1\r\nHost: localhost\r\nUser-Agent: Mozilla/5.0 fscan/1.8\r\nConnection: close\r\n\r\n";
    let mut resp = [0u8; 1024];
    let n = tcp_request(ip, port, req.as_ptr(), req.len(), resp.as_mut_ptr(), resp.len(), 1500);
    if n > 0 {
        let r_str = unsafe { core::str::from_utf8_unchecked(&resp[..n as usize]) };
        if let Some(title) = extract_html_title(r_str) { report_info(ip, port, "Title", title); }
        if let Some(server) = find_header(r_str, "Server: ") { report_info(ip, port, "Server", server); }
        if r_str.contains("rememberMe=deleteMe") { report_vuln(ip, port, "Shiro", "Key mismatch or default detected"); }
        if r_str.contains("Whitelabel") || r_str.contains("timestamp") { report_info(ip, port, "App", "SpringBoot"); }
    }
}

fn poc_weblogic_t3(ip: u32, port: u16) {
    let req = b"t3 12.2.1\nAS:255\nHL:19\nMS:10000000\n\n";
    let mut resp = [0u8; 64];
    let n = tcp_request(ip, port, req.as_ptr(), req.len(), resp.as_mut_ptr(), resp.len(), 1500);
    if n > 0 {
        let r_str = unsafe { core::str::from_utf8_unchecked(&resp[..n as usize]) };
        if r_str.contains("HELO:") { report_vuln(ip, port, "WebLogic", "T3 Protocol Enabled"); }
    }
}

fn poc_ms17_010(ip: u32, port: u16) {
    let req = [
        0x00, 0x00, 0x00, 0x85, 0xff, 0x53, 0x4d, 0x42, 0x72, 0x00, 0x00, 0x00, 0x00, 0x18, 0x53, 0xc8,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xfe,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x62, 0x00, 0x02, 0x50, 0x43, 0x20, 0x4e, 0x45, 0x54, 0x57, 0x4f,
        0x52, 0x4b, 0x20, 0x50, 0x52, 0x4f, 0x47, 0x52, 0x41, 0x4d, 0x20, 0x31, 0x2e, 0x30, 0x00, 0x02,
        0x4e, 0x54, 0x20, 0x4c, 0x4d, 0x20, 0x30, 0x2e, 0x31, 0x32, 0x00
    ];
    let mut resp = [0u8; 64];
    let n = tcp_request(ip, port, req.as_ptr(), req.len(), resp.as_mut_ptr(), resp.len(), 1800);
    if n > 35 && resp[4] == 0xff && resp[5] == b'S' && resp[6] == b'M' && resp[7] == b'B' {
        report_vuln(ip, port, "MS17-010", "SMBv1 found, risk of EternalBlue");
    }
}

fn poc_redis_unauth(ip: u32, port: u16) {
    let req = b"info\r\n";
    let mut resp = [0u8; 256];
    let n = tcp_request(ip, port, req.as_ptr(), req.len(), resp.as_mut_ptr(), resp.len(), 1000);
    if n > 0 {
        let r_str = unsafe { core::str::from_utf8_unchecked(&resp[..n as usize]) };
        if r_str.contains("redis_version") { report_vuln(ip, port, "Redis", "Unauthorized Found"); }
    }
}

fn poc_mysql_banner(ip: u32, port: u16) {
    let mut resp = [0u8; 128];
    let n = tcp_request(ip, port, core::ptr::null(), 0, resp.as_mut_ptr(), resp.len(), 1200);
    if n > 5 && resp[4] >= 10 { report_info(ip, port, "DB", "MySQL protocol detected"); }
}

fn poc_elasticsearch_unauth(ip: u32, port: u16) {
    let req = b"GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n";
    let mut resp = [0u8; 256];
    let n = tcp_request(ip, port, req.as_ptr(), req.len(), resp.as_mut_ptr(), resp.len(), 1200);
    if n > 0 {
        let r_str = unsafe { core::str::from_utf8_unchecked(&resp[..n as usize]) };
        if r_str.contains("cluster_name") || r_str.contains("lucene") { report_vuln(ip, port, "ES", "Elasticsearch Found"); }
    }
}

// --- Internal ---

fn tcp_request(ip: u32, port: u16, send: *const u8, s_len: usize, recv: *mut u8, r_len: usize, timeout: u32) -> usize {
    let params = TcpSendRecvParams {
        target_ip: ip, port, timeout_ms: timeout,
        send_ptr: send as u32, send_len: s_len as u32,
        recv_ptr: recv as u32, recv_len: r_len as u32,
    };
    unsafe { host_call(HASH_TCP_SEND_RECV, &params as *const _ as *const u8) as usize }
}

fn check_port(ip: u32, port: u16, timeout: u32) -> bool {
    let params = TcpConnectParams { target_ip: ip, port, timeout_ms: timeout };
    unsafe { host_call(HASH_TCP_CONNECT, &params as *const _ as *const u8) != 0 }
}

fn report(msg: &str) { unsafe { host_report_result(msg.as_ptr(), msg.len() as u32); } }
fn report_open(ip: u32, port: u16) { report("[+] "); report_ip(ip); report(":"); report_u16(port); report(" OPEN\n"); }
fn report_info(ip: u32, port: u16, k: &str, v: &str) {
    report_ip(ip); report(":"); report_u16(port); report(" ["); report(k); report("] "); report(v); report("\n");
}
fn report_vuln(ip: u32, port: u16, n: &str, d: &str) {
    report_ip(ip); report(":"); report_u16(port); report(" [!!!] VULN: "); report(n); report(" ("); report(d); report(")\n");
}

fn report_ip(ip: u32) {
    let mut buf = [0u8; 16];
    let mut idx = 0;
    idx += format_u8(&mut buf[idx..], (ip >> 24) as u8); buf[idx]=b'.'; idx+=1;
    idx += format_u8(&mut buf[idx..], (ip >> 16) as u8); buf[idx]=b'.'; idx+=1;
    idx += format_u8(&mut buf[idx..], (ip >> 8) as u8); buf[idx]=b'.'; idx+=1;
    idx += format_u8(&mut buf[idx..], ip as u8);
    report(unsafe { core::str::from_utf8_unchecked(&buf[..idx]) });
}

fn report_u16(n: u16) {
    let mut buf = [0u8; 6];
    let idx = format_u16(&mut buf, n);
    report(unsafe { core::str::from_utf8_unchecked(&buf[..idx]) });
}

fn format_u8(buf: &mut [u8], mut n: u8) -> usize {
    if n == 0 { buf[0] = b'0'; return 1; }
    let mut tmp = [0u8; 3]; let mut len = 0;
    while n > 0 { tmp[len] = b'0' + (n % 10); n /= 10; len += 1; }
    for i in 0..len { buf[i] = tmp[len - 1 - i]; }
    len
}

fn format_u16(buf: &mut [u8], mut n: u16) -> usize {
    if n == 0 { buf[0] = b'0'; return 1; }
    let mut tmp = [0u8; 5]; let mut len = 0;
    while n > 0 { tmp[len] = b'0' + (n % 10) as u8; n /= 10; len += 1; }
    for i in 0..len { buf[i] = tmp[len - 1 - i]; }
    len
}

fn extract_json_val<'a>(json: &'a str, key: &str) -> Option<&'a str> {
    if let Some(pos) = json.find(key) {
        let after_key = &json[pos + key.len()..];
        if let Some(start) = after_key.find(':') {
            let s_quote = after_key[start..].find('"')?;
            let sq = start + s_quote + 1;
            if let Some(end) = after_key[sq..].find('"') { return Some(&after_key[sq..sq+end]); }
        }
    }
    None
}

fn parse_cidr(s: &str) -> Option<(u32, u8)> {
    let slash = s.find('/');
    let (ip_str, mask) = if let Some(p) = slash { (&s[..p], u8::from_str(&s[p+1..]).ok()?) } else { (s, 32) };
    let mut ip = 0u32; let mut count = 0;
    for part in ip_str.split('.') { if count < 4 { ip = (ip << 8) | (u8::from_str(part).ok()? as u32); count += 1; } }
    if count == 4 { Some((ip, mask)) } else { None }
}

fn extract_html_title(html: &str) -> Option<&str> {
    if let Some(s) = html.find("<title>") {
        let start = s + 7;
        if let Some(e) = html[start..].find("</title>") { return Some(&html[start..start + e]); }
    }
    None
}

fn find_header<'a>(resp: &'a str, h: &str) -> Option<&'a str> {
    if let Some(pos) = resp.find(h) {
        let start = pos + h.len();
        if let Some(end) = resp[start..].find("\r\n") { return Some(&resp[start..start + end]); }
    }
    None
}

#[panic_handler]
fn panic(_info: &PanicInfo) -> ! { loop {} }
