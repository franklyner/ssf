package server

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Config property names
const (
	ConfigDBURI     = "db_uri"
	ConfigDBMaxConn = "db_max_connections"
)

// Repository provides access to the DB and prepared statements
type Repository struct {
	DB       *gorm.DB
	dbConfig DBconfig
}

// DBconfig holds all relevant db configurations
type DBconfig struct {
	DbURI              string
	DbMaxDBConnections int
}

// CreateRepository intizializes a repository
func CreateRepository(c *Config) *Repository {
	dbConf := readDBConfig(c)
	repo := Repository{
		dbConfig: dbConf,
	}
	err := repo.init()
	if err != nil {
		panic(err)
	}
	return &repo
}

func readDBConfig(c *Config) DBconfig {
	dbConfig := DBconfig{
		DbURI:              c.Get(ConfigDBURI),
		DbMaxDBConnections: c.GetInt(ConfigDBMaxConn),
	}

	return dbConfig
}

// Init initializes the repository incl. DB connection
func (r *Repository) init() error {
	gdb, err := gorm.Open(mysql.Open(r.dbConfig.DbURI), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open DB connection: %w", err)
	}
	r.DB = gdb

	db, err := gdb.DB()
	if err != nil {
		return fmt.Errorf("failed to retrieve underlying sqldb object: %w", err)
	}
	db.SetMaxOpenConns(r.dbConfig.DbMaxDBConnections)

	return nil
}

// RunMigrations migrates all provided entities which must be gorm compatible entities
func (r *Repository) RunMigrations(entities []interface{}) {
	for _, e := range entities {
		r.DB.AutoMigrate(e)
	}
}
