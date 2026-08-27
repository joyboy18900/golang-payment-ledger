package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang-payment-ledger/handler"
	"golang-payment-ledger/logs"
	"golang-payment-ledger/repository"
	"golang-payment-ledger/service"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	initConfig()
	runMigrations()

	db := openGormDB()

	accountRepo := repository.NewAccountRepositoryDB(db)
	accountSvc := service.NewAccountService(accountRepo)
	accountHdlr := handler.NewAccountHandler(accountSvc)

	transactionRepo := repository.NewTransactionRepositoryDB(db)
	transferSvc := service.NewTransferService(transactionRepo)
	transferHdlr := handler.NewTransferHandler(transferSvc)

	app := fiber.New()
	app.Post("/accounts", accountHdlr.Create)
	app.Get("/accounts/:id/balance", accountHdlr.GetBalance)
	app.Post("/transfer", transferHdlr.Transfer)

	port := viper.GetString("app.port")
	logs.Info("server started on port " + port)
	if err := app.Listen(":" + port); err != nil {
		logs.Error(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("read config: %w", err))
	}
}

func postgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		viper.GetString("db.user"),
		viper.GetString("db.password"),
		viper.GetString("db.host"),
		viper.GetInt("db.port"),
		viper.GetString("db.name"),
		viper.GetString("db.sslmode"),
	)
}

func openGormDB() *gorm.DB {
	db, err := gorm.Open(postgres.Open(postgresDSN()), &gorm.Config{
		TranslateError: true,
		Logger:         gormLogger(),
	})
	if err != nil {
		panic(fmt.Errorf("open postgres: %w", err))
	}

	return db
}

func gormLogger() logger.Interface {
	return logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
	})
}

func runMigrations() {
	dsn := strings.Replace(postgresDSN(), "postgres://", "pgx5://", 1)

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		panic(fmt.Errorf("new migrate: %w", err))
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		panic(fmt.Errorf("migrate up: %w", err))
	}

	logs.Info("migrations up to date")
}
