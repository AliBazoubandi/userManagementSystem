package connection

import (
	"context"
	"fmt"
	"main/config"
	"main/utility"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type DataBase struct {
	Conn *pgxpool.Pool
}

var DB DataBase

func InitDatabase() {
	// Load logger
	logger := utility.AppLogger.Logger

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", config.AppConfig.Database.Host, config.AppConfig.Database.Port, config.AppConfig.Database.User, config.AppConfig.Database.Password, config.AppConfig.Database.DataBase, config.AppConfig.Database.SSLMode)
	ctx := context.Background()
	conf, err := pgxpool.ParseConfig(psqlInfo)
	if err != nil {
		logger.Error("Fatal Error Connection To Data Base", zap.Error(err))
	}

	conf.MaxConns = 20
	conf.MinConns = 2
	conf.MaxConnIdleTime = 0

	conn, err := pgxpool.NewWithConfig(ctx, conf)
	if err != nil {
		logger.Error("Fatal Error Connecting To Database", zap.Error(err))
	}

	DB.Conn = conn

	if err = DB.Conn.Ping(ctx); err != nil {
		logger.Error("There Is A Problem In Connection", zap.Error(err))
	}
	logger.Info("Connection To Data Base Established")
}

func CloseDB() {
	if DB.Conn != nil {
		DB.Conn.Close()
		utility.AppLogger.Logger.Info("Connection To Data Base Closed")
	}
}
