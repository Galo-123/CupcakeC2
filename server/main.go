package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"cupcake-server/controllers"
	"cupcake-server/pkg/config"
	"cupcake-server/pkg/middleware"
	"cupcake-server/pkg/store"
	"cupcake-server/services"
)

//go:embed dist/*
var embeddedFiles embed.FS

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	store.InitDB()
	store.ResetAllAgentsOffline()
	go services.RestoreListeners()
	go services.RestoreTunnels()

	gin.SetMode(gin.ReleaseMode)
	adminRouter := gin.New()
	adminRouter.Use(gin.Logger(), gin.Recovery())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	adminRouter.Use(cors.New(corsConfig))

	adminRouter.Use(middleware.AuthMiddleware())

	api := adminRouter.Group("/api")
	{
		api.GET("/dashboard", controllers.GetDashboard)
		api.GET("/clients", controllers.GetClients)
		api.GET("/clients/history/:uuid", controllers.HandleGetAgentHistory)
		api.DELETE("/clients/:uuid", controllers.DeleteClient)
		api.POST("/clients/migrate", controllers.MigrateClient)
		api.POST("/cmd", controllers.SendCommand)
		api.GET("/resp", controllers.GetResponse)

		api.GET("/listeners", controllers.ListListeners)
		api.POST("/listeners", controllers.CreateListener)
		api.POST("/listeners/:id/stop", controllers.StopListener)
		api.POST("/listeners/:id/start", controllers.StartListener)
		api.DELETE("/listeners/:id", controllers.DeleteListener)

		api.POST("/tunnel/start", controllers.StartTunnel)
		api.POST("/tunnel/stop", controllers.StopTunnel)
		api.POST("/tunnel/delete", controllers.DeleteTunnelController)
		api.GET("/tunnel", controllers.ListTunnels)

		api.POST("/socks/start", controllers.StartSocks)
		api.POST("/socks/stop", controllers.StopSocks)
		api.POST("/socks/delete", controllers.DeleteTunnelController)
		api.GET("/socks", controllers.ListSocks)

		files := api.Group("/files")
		{
			files.GET("/list", controllers.ListFilesController)
			files.GET("/read", controllers.ReadFileController)
			files.POST("/delete", controllers.DeleteFilesController)
			files.POST("/upload", controllers.Upload)
			files.POST("/download", controllers.HandleFsDownload)
		}

		processes := api.Group("/processes")
		{
			processes.GET("/list", controllers.ListProcesses)
			processes.POST("/kill", controllers.KillProcess)
		}

		api.GET("/shell/:uuid", controllers.HandleAdminShell)
		api.GET("/pty/:uuid", controllers.StreamPTY)

		plugins := api.Group("/plugins")
		{
			plugins.GET("", controllers.HandleListPlugins)
			plugins.POST("/run", controllers.HandleRunPlugin)
			plugins.POST("/upload", controllers.HandleUploadPlugin)
			plugins.DELETE("/:id", controllers.HandleDeletePlugin)
			plugins.GET("/result/:task_id", controllers.HandleGetPluginResult)
		}

		api.GET("/build/logs/:task_id", controllers.HandleBuildLogsWS)

		transfer := api.Group("/transfer")
		{
			services.InitTransfer()
			transfer.POST("/upload", services.HandleAgentUpload)
			transfer.GET("/download/:filename", services.HandleAgentDownload)
			transfer.Static("/static", "./storage/public_tools")
		}

		settings := api.Group("/settings")
		{
			settings.GET("/users", controllers.HandleGetUsers)
			settings.POST("/users", controllers.HandleAddUser)
			settings.PUT("/users/:id", controllers.HandleUpdateUser)
			settings.DELETE("/users/:id", controllers.HandleDeleteUser)
			settings.GET("/logs/login", controllers.HandleGetLoginLogs)
			settings.GET("/config", controllers.HandleGetSettings)
			settings.POST("/config", controllers.HandleUpdateSettings)
			settings.GET("/webhooks", controllers.HandleGetWebhooks)
			settings.POST("/webhooks", controllers.HandleSaveWebhook)
			settings.DELETE("/webhooks/:id", controllers.HandleDeleteWebhook)
		}

		api.POST("/generate", controllers.HandleGenerate)
		api.GET("/generate/stream", controllers.HandleGenerateStream)
		api.Static("/downloads", "./storage/payloads")

		api.POST("/auth/login", controllers.HandleLogin)
		api.POST("/auth/logout", func(c *gin.Context) { c.JSON(200, gin.H{"msg": "logged out"}) })
		api.POST("/maintenance/reset", controllers.HandleMaintenanceReset)
		api.GET("/maintenance/export", controllers.HandleMaintenanceExport)
	}

	distFS, _ := fs.Sub(embeddedFiles, "dist")
	staticServer := http.FileServer(http.FS(distFS))

	adminRouter.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		cloakTarget := store.GetSetting("opsec_cloak_url")
		if cloakTarget != "" && !strings.HasPrefix(path, "/api/") {
			c.Redirect(http.StatusFound, cloakTarget)
			return
		}
		if strings.HasPrefix(path, "/api/") {
			c.JSON(404, gin.H{"error": "API not found"})
			return
		}
		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" { cleanPath = "index.html" }
		f, err := distFS.Open(cleanPath)
		if err == nil {
			f.Close()
			staticServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		index, _ := distFS.Open("index.html")
		defer index.Close()
		stat, _ := index.Stat()
		c.DataFromReader(200, stat.Size(), "text/html; charset=utf-8", index, nil)
	})

	banner := "\x1b[36m" + `
    ______  __    __  .______     ______      ___       __  ___  _______   
   /      ||  |  |  | |   _  \   /      |    /   \     |  |/  / |   ____|  
  |  ,----'|  |  |  | |  |_)  | |  ,----'   /  ^  \    |  '  /  |  |__     
  |  |     |  |  |  | |   ___/  |  |       /  /_\  \   |    <   |   __|    
  |  ` + "`" + `----.|  ` + "`" + `--'  | |  |      |  ` + "`" + `----. /  _____  \  |  .  \  |  |____   
   \______| \______/  | _|       \______|/__/     \__\ |__|\__\ |_______|  
` + "\x1b[0m" + `                                                                         
                          >> MCP AUTOMATION READY <<                   
`
	log.Println(banner)
	log.Printf("   Cupcake C2 控制终端                   ")
	log.Printf("   Web UI: http://127.0.0.1:%d         ", cfg.AdminPort)
	log.Println("-----------------------------------------")

	if err := adminRouter.Run(fmt.Sprintf(":%d", cfg.AdminPort)); err != nil {
		log.Fatal(err)
	}
}
