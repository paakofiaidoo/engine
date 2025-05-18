package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ApiConfig interface {
	LoadEnv()
	DBConfig() *gorm.Config
	ServerAddr() string
	IsDevelopment() bool
	IsProduction() bool
}

type apiConfig struct {
}

func (cfg *apiConfig) IsDevelopment() bool {
	return os.Getenv("APP.ENV") == "development"
}

func (cfg *apiConfig) IsProduction() bool {
	return os.Getenv("APP.ENV") == "production"
}

func (cfg *apiConfig) ServerAddr() string {
	return fmt.Sprintf("%s:%s", os.Getenv("APP.HOST"), os.Getenv("APP.PORT"))
}

func (cfg *apiConfig) LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("failed to load .env")
	}
	log.Println(".env loaded successfully")
}

func (cfg *apiConfig) DBConfig() *gorm.Config {
	return &gorm.Config{
		SkipDefaultTransaction:                   false,
		NamingStrategy:                           nil,
		FullSaveAssociations:                     false,
		Logger:                                   logger.Default.LogMode(logger.Info),
		NowFunc:                                  nil,
		DryRun:                                   false,
		PrepareStmt:                              false,
		DisableAutomaticPing:                     false,
		DisableForeignKeyConstraintWhenMigrating: false,
		IgnoreRelationshipsWhenMigrating:         false,
		DisableNestedTransaction:                 false,
		AllowGlobalUpdate:                        false,
		QueryFields:                              false,
		CreateBatchSize:                          0,
		TranslateError:                           false,
		ClauseBuilders:                           nil,
		ConnPool:                                 nil,
		Dialector:                                nil,
		Plugins:                                  nil,
	}
}

func New() ApiConfig {
	return &apiConfig{}
}
