package main

import (
	"demo/network/go_server/app/controllers"
	"demo/network/go_server/app/models"
	"demo/network/go_server/app/repositories"
	"demo/network/go_server/app/seeders"
	"demo/network/go_server/app/services"
	"demo/network/go_server/config"
	"demo/network/go_server/global"
	"demo/network/go_server/server"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	MSG_LOGIN_REQ  = 0xA1
	MSG_DEVICE_REQ = 0xC1
)

func main() {
	fmt.Println("--- MSG-ROUTER GO SERVER ---")

	// Load Config
	if err := config.LoadConfig("config.yml"); err != nil {
		log.Fatalf("[Error] Failed to load config.yml: %v", err)
	}

	// Init DB (MySQL)
	// User: root, Pass: root, DB: sagiri_guard
	dsn := config.AppConfig.Server.DBDSN
	fmt.Println("[Init] Connecting to MySQL...")

	dbConn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Printf("[Error] DB Connection failed: %v", err)
		log.Println("Ensure database 'sagiri_guard' exists and MySQL is running.")
		return // Exit if DB fails
	}
	global.DB = dbConn

	if global.DB != nil {
		// Auto Migrate
		global.DB.AutoMigrate(
			&models.User{},
			&models.Device{},
			&models.Log{},
			&models.Command{},
			&models.FirewallCategory{},
			&models.FirewallDomain{},
			&models.DeviceConfig{},
			&models.DeviceFileHistory{},
			&models.FileNode{},
			&models.BackupSession{},
			&models.BackupSnapshot{},
		)

		// Seed Admin
		var count int64
		global.DB.Model(&models.User{}).Where("username = ?", "admin").Count(&count)
		if count == 0 {
			global.DB.Create(&models.User{Username: "admin", Password: "admin"})
			fmt.Println("Seeded admin user.")
		}

		// seed user
		global.DB.Model(&models.User{}).Where("username = ?", "user").Count(&count)
		if count == 0 {
			global.DB.Create(&models.User{Username: "user", Password: "user"})
			fmt.Println("Seeded user user.")
			fmt.Println("Seeded user user.")
		}

		// Seed Firewall
		if err := seeders.SeedFirewall(global.DB); err != nil {
			log.Printf("[Warning] Firewall Seeding failed: %v", err)
		} else {
			fmt.Println("Firewall Seeded successfully.")
		}
	}

	// Register Routes
	// Register Routes
	server.Router[MSG_LOGIN_REQ] = controllers.HandleLogin
	server.Router[MSG_DEVICE_REQ] = controllers.HandleDeviceRegister
	server.Router[0xB1] = controllers.HandleListUsers

	// --- MVC INITIALIZATION ---
	// 1. Repositories
	cmdRepo := repositories.NewCommandRepository(global.DB)
	devRepo := repositories.NewDeviceRepository(global.DB)
	fwRepo := repositories.NewFirewallRepository(global.DB)
	logRepo := repositories.NewLogRepository(global.DB)
	histRepo := repositories.NewFileHistoryRepository(global.DB)
	nodeRepo := repositories.NewFileNodeRepository(global.DB)
	backupRepo := repositories.NewBackupRepository(global.DB)

	// 2. Services
	// CommandSvc depends on CommandRepo
	cmdSvc := services.NewCommandService(cmdRepo)

	// LogSvc depends on LogRepo
	logSvc := services.NewLogService(logRepo)

	// FirewallSvc depends on DeviceRepo, FirewallRepo, CommandSvc
	fwSvc := services.NewFirewallService(devRepo, fwRepo, cmdSvc)

	// AdminSvc depends on CommandRepo, CommandSvc
	adminSvc := services.NewAdminService(cmdRepo, cmdSvc)

	// FileHistorySvc
	histSvc := services.NewFileHistoryService(histRepo, nodeRepo)

	// DirectoryTreeSvc
	treeSvc := services.NewDirectoryTreeService(nodeRepo)

	// BackupSvc
	backupSvc := services.NewBackupService(backupRepo)

	// 3. Inject into Controllers
	controllers.Init(fwSvc, adminSvc, logSvc, histSvc, treeSvc, backupSvc)
	controllers.SetDirectoryTreeService(treeSvc)
	controllers.SetBackupService(backupSvc)

	fmt.Println("[Init] MVC Layer Initialized.")
	// --------------------------

	// Admin Flow
	server.Router[0xD6] = controllers.HandleAdminLogin
	server.Router[0xD1] = controllers.HandleAdminGetLogs
	server.Router[0xD4] = controllers.HandleClientLogUpload
	server.Router[0xD8] = controllers.HandleAdminGetStoredLogs
	server.Router[0xDA] = controllers.HandleAdminGetCommandHistory
	server.Router[0xDB] = controllers.HandleAdminGetCommandHistory // Actually response type usually not routed, but for consistency if reused.
	server.Router[0xE1] = controllers.HandleAdminFirewallControl
	server.Router[0xE4] = controllers.HandleClientGetFirewallConfig
	server.Router[0xE6] = controllers.HandleClientFileSync   // New Route
	server.Router[0xE8] = controllers.HandleAdminGetFileTree // New Route for Admin Tree
	server.Router[0xF1] = controllers.HandleBackupInit
	server.Router[0xF3] = controllers.HandleBackupChunk
	server.Router[0xF5] = controllers.HandleBackupFinish
	server.Router[0xF7] = controllers.HandleBackupCancel

	server.Init(config.AppConfig.Server.Port, config.AppConfig.Server.APIPort)
	server.SetHandler(server.GoRequestHandler)

	fmt.Printf("[Server] Starting on Ports %d (Notification) and %d (API)...\n", config.AppConfig.Server.Port, config.AppConfig.Server.APIPort)
	server.Start()

	// Block main thread
	select {}
}
