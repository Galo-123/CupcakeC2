package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"cupcake-server/pkg/globals"
	"cupcake-server/pkg/hub"
	"cupcake-server/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
)

// upgrader 使用 globals 包中定义的全局实例，避免重复定义
var upgrader = globals.Upgrader

func StreamPTY(c *gin.Context) {
	uuid := c.Param("uuid")
	val, ok := globals.Clients.Load(uuid)
	if !ok {
		c.JSON(404, gin.H{"error": "Agent offline"})
		return
	}
	client := val.(*globals.Client)

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil { return }
	defer ws.Close()

	if client.YamuxSession == nil {
		StreamPTYFallback(ws, client)
		return
	}

	stream, err := client.YamuxSession.Open()
	if err != nil { return }
	defer stream.Close()

	if _, err := stream.Write([]byte{0x01}); err != nil { return }

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			mt, msg, err := ws.ReadMessage()
			if err != nil { return }
			if mt == websocket.TextMessage || mt == websocket.BinaryMessage {
				stream.Write(msg)
			}
		}
	}()

	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if err != nil { break }
		ws.WriteMessage(websocket.BinaryMessage, buf[:n])
	}
}

func StreamPTYFallback(ws *websocket.Conn, client *globals.Client) {
	doneToken := "__CUPCAKE_DONE__"
	modePacket := map[string]string{
		"type":    "PTY_MODE",
		"content": "fallback",
	}
	if data, err := json.Marshal(modePacket); err == nil {
		ws.WriteMessage(websocket.TextMessage, data)
	}
	if data, err := json.Marshal(map[string]string{"type": "PTY_DONE"}); err == nil {
		ws.WriteMessage(websocket.TextMessage, data)
	}
	isWindows := strings.Contains(strings.ToLower(client.OS), "windows")
	if _, loaded := globals.PTYState.LoadOrStore(client.UUID, true); !loaded {
		startMsg := globals.MessageWrapper{
			MsgType: "command",
			Payload: globals.CommandPayload{
				CommandType:    "shell_interactive",
				CommandContent: "",
				ReqID:          uuid.New().String(),
			},
		}
		_ = services.WriteEncryptedMessage(client, startMsg)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for output := range client.OutputChannel {
			ws.WriteMessage(websocket.TextMessage, []byte(output))
		}
	}()
	lineBuf := make([]rune, 0, 256)
	flushLine := func() {
		if len(lineBuf) == 0 {
			return
		}
		line := string(lineBuf)
		if strings.TrimSpace(line) != "" {
			cmd := line
			if isWindows {
				clean := strings.TrimSpace(line)
				if !strings.HasPrefix(clean, "@") {
					clean = "@" + clean
				}
				cmd = fmt.Sprintf("%s & @echo %s", clean, doneToken)
			} else {
				cmd = fmt.Sprintf("%s; echo %s", line, doneToken)
			}
			client.CommandChannel <- cmd
		}
		lineBuf = lineBuf[:0]
	}
	for {
		mt, msg, err := ws.ReadMessage()
		if err != nil { break }
		if mt == websocket.TextMessage || mt == websocket.BinaryMessage {
			for _, r := range string(msg) {
				switch r {
				case '\r', '\n':
					flushLine()
				case 0x7f, 0x08:
					if len(lineBuf) > 0 {
						lineBuf = lineBuf[:len(lineBuf)-1]
					}
				default:
					if r < 0x20 {
						continue
					}
					lineBuf = append(lineBuf, r)
				}
			}
		}
	}
}

func HandleAdminShell(c *gin.Context) {
	uuid := c.Param("uuid")
	val, ok := globals.Clients.Load(uuid)
	if !ok {
		c.JSON(404, gin.H{"error": "Agent Offline"})
		return
	}
	client := val.(*globals.Client)

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil { return }
	defer ws.Close()

	// 🛡️ Anti-DoS: 限制管理员 Shell WebSocket 单帧大小为 1MB（命令不应超过此大小）
	ws.SetReadLimit(1 * 1024 * 1024)

	go func() {
		for output := range client.OutputChannel {
			var packet hub.WsPacket
			if err := json.Unmarshal([]byte(output), &packet); err != nil {
				packet = hub.WsPacket{MsgType: "TERM", Content: output}
			}
			ws.WriteJSON(packet)
		}
	}()

	for {
		var msg hub.WsPacket
		if err := ws.ReadJSON(&msg); err != nil { break }
		client.CommandChannel <- msg.Content
	}
}

func MigrateClient(c *gin.Context) {
	var req struct {
		UUID   string `json:"uuid"`
		Target string `json:"target"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid input"})
		return
	}
	if err := services.MigrateToMemory(req.UUID, req.Target); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

func SendCommand(c *gin.Context) {
	var req struct {
		UUID    string `json:"uuid"`
		Command string `json:"cmd"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid input"})
		return
	}
	if err := services.SendCommand(req.UUID, req.Command); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

func HandleConnectBindAgent(c *gin.Context) {
	var req struct {
		TargetAddr string `json:"target_addr"`
		ListenerID string `json:"listener_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	val, ok := globals.Listeners.Load(req.ListenerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listener not found"})
		return
	}
	ln := val.(*globals.Listener)

	if err := services.ConnectToBindAgent(req.TargetAddr, ln); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "Connecting to bind agent..."})
}

func GetResponse(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
