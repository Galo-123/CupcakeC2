/**
 * Intranet Commander - Defensive Audit Orchestrator
 * 技能入口：输出审计清单/整改建议，并对高风险/破坏性动作做拦截。
 *
 * 设计目标：
 * - 默认只读、最小影响
 * - 禁止规避/对抗安全控制、清理日志、建立持久化、破坏性操作
 */

const BASELINE_COMMANDS = {
    windows: {
        identity: "whoami /all",
        os_info:
            "ver && reg query \"HKLM\\Software\\Microsoft\\Windows NT\\CurrentVersion\" /v ProductName && reg query \"HKLM\\Software\\Microsoft\\Windows NT\\CurrentVersion\" /v CurrentBuildNumber && reg query \"HKLM\\Software\\Microsoft\\Windows NT\\CurrentVersion\" /v UBR",
        network: "ipconfig /all && route print",
        firewall: "netsh advfirewall show allprofiles",
        hotfixes:
            "powershell -NoProfile -Command \"Get-HotFix | Sort-Object InstalledOn -Descending | Select-Object -First 20 | Format-Table -AutoSize\"",
        defender_status:
            "powershell -NoProfile -Command \"if (Get-Command Get-MpComputerStatus -ErrorAction SilentlyContinue) { Get-MpComputerStatus | Select-Object AMServiceEnabled,AntispywareEnabled,RealTimeProtectionEnabled,IoavProtectionEnabled,NISEnabled | Format-List } else { \\\"Defender cmdlets not available.\\\" }\"",
        audit_policy_hint: "auditpol /get /category:*"
    },
    linux: {
        identity: "id",
        os_info: "cat /etc/os-release 2>/dev/null || uname -a",
        network: "ip addr 2>/dev/null || ifconfig -a 2>/dev/null",
        listening_ports: "ss -lntup 2>/dev/null || netstat -lntup 2>/dev/null",
        time_sync: "timedatectl status 2>/dev/null || date",
        audit_policy_hint: "auditctl -s 2>/dev/null || echo \"auditd not installed/permission denied\""
    }
};

const FORBIDDEN_PATTERNS = [
    // Destructive / stability-impacting
    /\brm\s+-rf\s+\/(?:\s|$)/i,
    /\brm\s+-rf\s+\/\*/i,
    /\bmkfs(\.|_)?\w*\b/i,
    /\b(?:dd|shred)\b/i,
    /\bdiskpart\b/i,
    /\bformat\s+\w*:\b/i,
    /\bshutdown\b/i,
    /\breboot\b/i,
    /\binit\s+[06]\b/i,

    // Log tampering / trace removal
    /\bwevtutil\s+cl\b/i,
    /\bClear-EventLog\b/i,
    /\bdel\b.*\\winevt\\logs\\?/i,
    /\bexport\s+HISTFILE\s*=\s*\/dev\/null\b/i,
    /\bhistory\s+-c\b/i,

    // Security control disablement
    /\bSet-MpPreference\b.*Disable/i,
    /\bnetsh\b.*\bstate\s+off\b/i,
    /\bufw\s+disable\b/i,

    // Persistence / backdoor creation (block common verbs)
    /\bschtasks\b.*\bcreate\b/i,
    /\breg\b.*\badd\b/i,
    /\bsc\b.*\bcreate\b/i,
    /\bcrontab\b.*\b-e\b/i,
    /\bsystemctl\b.*\benable\b/i,

    // Offensive tooling / credential theft indicators (defensive block)
    /\bmimikatz\b/i,
    /\brubeus\b/i,
    /\bsecretsdump\b/i,
    /\bpsexec\b/i,
    /\bchisel\b/i,
    /\bfrp\b/i,
    /\bsharphound\b/i
];

class IntranetCommander {
    constructor() {
        this.version = "1.1.0";
        this.mode = "Audit";
    }

    /**
     * 兼容旧接口：返回平台的“只读基线采集”命令集合。
     */
    getSilentDiscovery(platform = "windows") {
        return this.getBaselineDiscovery(platform);
    }

    getBaselineDiscovery(platform = "windows") {
        return BASELINE_COMMANDS[platform] || {};
    }

    /**
     * 执行前审计：阻止破坏性/规避/对抗类动作。
     * 注意：这是启发式拦截，不替代正式变更评审与权限控制。
     */
    auditAction(cmd) {
        const normalized = String(cmd ?? "").trim();
        if (!normalized) return { allowed: false, reason: "空指令。" };

        for (const pattern of FORBIDDEN_PATTERNS) {
            if (pattern.test(normalized)) {
                return {
                    allowed: false,
                    reason:
                        "该指令疑似破坏性、痕迹清理、关闭安全控制或建立持久化，已按审计/加固版规则拦截。"
                };
            }
        }

        return { allowed: true };
    }

    /**
     * 旧接口保留但语义调整：从“逃逸建议”改为“隔离与加固检查清单”。
     */
    getEscapeStrategy(type) {
        return this.getHardeningChecklist(type);
    }

    getHardeningChecklist(type) {
        switch (type) {
            case "docker":
            case "container":
                return [
                    "容器/宿主机隔离检查：",
                    "- 避免使用特权容器（--privileged），最小化 capabilities",
                    "- 禁止将 /var/run/docker.sock 挂载到业务容器；需要时使用受控的代理/最小权限方案",
                    "- 限制 hostPath 挂载与宿主机 PID/Network namespace 共享",
                    "- 开启并验证 seccomp/AppArmor/SELinux、只读根文件系统、rootless 运行时（可行时）",
                    "- 镜像来源与签名：启用镜像扫描、SBOM、签名验证与准入策略",
                ].join("\n");
            case "k8s":
            case "kubernetes":
                return [
                    "K8s 加固检查：",
                    "- RBAC 最小权限；禁用默认 ServiceAccount 的 token 自动挂载（按需开启）",
                    "- 启用准入控制与策略（PSA/OPA/kyverno 等，按组织标准）",
                    "- 网络策略默认拒绝，按需放通；启用审计日志并接入 SIEM",
                    "- 节点与控制面补丁：定期升级并验证 CVE 修复",
                ].join("\n");
            case "vm":
            case "virtualization":
                return [
                    "虚拟化加固检查：",
                    "- 及时更新 hypervisor 与 Guest Additions/Tools，关闭不必要的共享功能（共享文件夹/剪贴板/拖放）",
                    "- 控制设备直通与管理面访问，开启日志与告警",
                    "- 明确隔离边界与东西向流量策略，避免“管理网”与“业务网”混用",
                ].join("\n");
            default:
                return "未定义类型。可选：docker/container、k8s/kubernetes、vm/virtualization。";
        }
    }
}

module.exports = new IntranetCommander();
