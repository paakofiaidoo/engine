package database

import (
	"fmt"
	"juki-engine/pkg/data/models"
	"log"
	"math"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"

	"gorm.io/gorm"
)

type Connection interface {
	Engine() *gorm.DB
	Connect()
	Migrate()
	SetConfig(*gorm.Config)
}

type gormDB struct {
	engine *gorm.DB
	cfg    *gorm.Config
	models []interface{}
}

func (db *gormDB) Engine() *gorm.DB {
	return db.engine
}

func New() Connection {
	return &gormDB{
		models: []interface{}{
			&models.Page{},
			&models.Project{},
			&models.File{},
			&models.Symbol{},
			&models.DraftCommand{},
			&models.Layout{},
			&models.ActivityLog{},
			&models.TerminalActivity{},
			&models.Content{},
		},
	}
}

func (db *gormDB) SetConfig(cfg *gorm.Config) {
	db.cfg = cfg
}

func (db *gormDB) Connect() {
	dbType := os.Getenv("DB.TYPE")
	log.Println("Loading db config for: ", dbType)
	var err error
	switch dbType {

	case "postgres":
		log.Println("loading config for postgres")
		db.engine, err = db.attemptPGConnection()
		if err != nil {
			log.Fatalln("unable to load postgres db", err.Error())
		}

		log.Println("Postgres db connected successfully")

	default:

		db.engine, err = db.loadSqliteDB()
		if err != nil {
			log.Fatalln("unable to load sqlite db", err.Error())
		}
	}

	db.Migrate()
}

func (db *gormDB) loadSqliteDB() (*gorm.DB, error) {
	dbName := fmt.Sprintf("%s.db?parseTime=True", os.Getenv("DB.NAME"))
	conn, err := gorm.Open(sqlite.Open(dbName), db.cfg)

	if err != nil {
		panic("failed to connect database")
		return nil, err
	}
	log.Println("connected to sqlite db")

	return conn, nil
}

func (db *gormDB) attemptPGConnection() (*gorm.DB, error) {
	var counts int64
	var backOff = 1 * time.Second
	var connection *gorm.DB

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s",
		os.Getenv("DB.HOST"),
		os.Getenv("DB.USER"),
		os.Getenv("DB.PASS"),
		os.Getenv("DB.NAME"),
		os.Getenv("DB.PORT"),
	)

	for {
		c, err := gorm.Open(postgres.Open(dsn), db.cfg)
		if err != nil {
			fmt.Println("Postgres DB not yet ready to connect!")
			counts++
		} else {
			fmt.Println("Connected to postgres db!")
			connection = c
			break
		}

		if counts > 5 {
			fmt.Println(err)
			return nil, err
		}

		backOff = time.Duration(math.Pow(float64(counts), 2)) * time.Second
		log.Println("Backing off...")
		time.Sleep(backOff)
	}

	return connection, nil
}

func (db *gormDB) Migrate() {
	err := db.engine.AutoMigrate(db.models...)
	if err != nil {
		log.Fatalln("migration failed:", err)
	}

	log.Println("tables migrated successfully!")
}
