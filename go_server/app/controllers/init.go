package controllers

import (
	"demo/network/go_server/app/services"
)

var (
	FirewallSvc    *services.FirewallService
	AdminSvc       *services.AdminService
	LogSvc         *services.LogService
	FileHistorySvc *services.FileHistoryService
	TreeSvc        *services.DirectoryTreeService
	BackupSvc      *services.BackupService
)

func Init(fSvc *services.FirewallService, aSvc *services.AdminService, lSvc *services.LogService, fhSvc *services.FileHistoryService, tSvc *services.DirectoryTreeService, bSvc *services.BackupService) {
	FirewallSvc = fSvc
	AdminSvc = aSvc
	LogSvc = lSvc
	FileHistorySvc = fhSvc
	TreeSvc = tSvc
	BackupSvc = bSvc
}

func SetBackupService(svc *services.BackupService) {
	BackupSvc = svc
}
