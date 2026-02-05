package controllers

import (
	"cupcake-server/pkg/globals"
	"cupcake-server/pkg/model"
	"cupcake-server/pkg/store"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

func GetDashboard(c *gin.Context) {
	v, _ := mem.VirtualMemory()
	cStats, _ := cpu.Percent(0, false)
	cpuPerc := 0.0
	if len(cStats) > 0 { cpuPerc = cStats[0] }
	dStats, _ := disk.Usage("/")
	hInfo, _ := host.Info()

	onlineCount := 0
	globals.Clients.Range(func(k, v interface{}) bool {
		onlineCount++
		return true
	})

	allAgents, _ := store.GetAllAgents()
	totalCount := len(allAgents)

	listenerCount := 0
	globals.Listeners.Range(func(k, v interface{}) bool {
		listenerCount++
		return true
	})

	templatesReady := true
	if _, err := os.Stat("assets/client_template_windows.exe"); os.IsNotExist(err) { templatesReady = false }
	if _, err := os.Stat("assets/client_template_linux"); os.IsNotExist(err) && templatesReady { templatesReady = false }

	c.JSON(http.StatusOK, gin.H{
		"cpu_usage":       fmt.Sprintf("%.1f", cpuPerc),
		"mem_usage":       fmt.Sprintf("%.1f", v.UsedPercent),
		"disk_usage":      fmt.Sprintf("%.1f", dStats.UsedPercent),
		"uptime":          hInfo.Uptime,
		"listener_count":  listenerCount,
		"client_count":    totalCount,
		"online_count":    onlineCount,
		"hostname":        hInfo.Hostname,
		"os":              hInfo.OS,
		"templates_ready": templatesReady,
	})
}

func GetClients(c *gin.Context) {
	agents, err := store.GetAllAgents()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if agents == nil {
		agents = []model.Agent{}
	}
	c.JSON(http.StatusOK, agents)
}

func HandleGetAgentHistory(c *gin.Context) {
	uuid := c.Param("uuid")
	history, _ := store.GetCommandHistory(uuid)
	c.JSON(http.StatusOK, history)
}

func DeleteClient(c *gin.Context) {
	uuid := c.Param("uuid")
	// Remove from memory
	globals.Clients.Delete(uuid)
	// Remove from database
	if err := store.DeleteAgent(uuid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
