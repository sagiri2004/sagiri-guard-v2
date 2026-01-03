package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

func Connect(cfg Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Error,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, err
	}

	// Auto Migrate
	// Note: We need to import models, but avoiding circular dependency if db imports models.
	// Ideally, migration happens in main or a separate initializer.
	// For now, let's just return the DB and let Main handle migration to keep this package clean
	// or use a separate Migrate function.
	// Revising plan: Let's keep Connect simple and return DB.
	// I entered this thinking I'd add AutoMigrate here, but I need to import `models`.
	// `db` package shouldn't depend on `models` if `models` might depend on `db` (though here models just use gorm).
	// Let's stick to the plan of "Main calls Migration".
	// Actually, to avoid import cycles, I'll pass the models to a Migrate function or just do it in main.
	// I will revert to just returning the DB instance as before, but I'll verifying the connection config.
	// Actually, the previous file content was fine.
	// I will use this step to just ADD the return.
	// Wait, the file content I saw earlier was correct.
	// I will instead create a new file `app/db/migrate.go` or just handle it in main.
	// Let's modify this tool call to just do nothing or improve the DSN if needed.
	// Actually, looking at the code, it returns `gorm.Open(...)`.
	// I will cancel this edit and instead do the migration in `main.go` as planned.
	return db, nil
}
