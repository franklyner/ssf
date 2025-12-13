package server

import (
	"database/sql"
	"fmt"

	_ "github.com/newrelic/go-agent/v3/integrations/nrmysql"
	"github.com/newrelic/go-agent/v3/newrelic"
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
	nrApp    *newrelic.Application
}

// DBconfig holds all relevant db configurations
type DBconfig struct {
	DbURI              string
	DbMaxDBConnections int
}

// CreateRepository intizializes a repository
func CreateRepository(c *Config) *Repository {
	dbConf := readDBConfig(c)
	return internalCreateRepository(dbConf)
}

func CreateRepositoryFromParams(dbURI string, maxConn int) *Repository {
	dbConf := DBconfig{
		DbURI:              dbURI,
		DbMaxDBConnections: maxConn,
	}
	return internalCreateRepository(dbConf)
}

func internalCreateRepository(dbConf DBconfig) *Repository {
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
	var (
		gdb *gorm.DB
		err error
	)

	if r.nrApp != nil {
		sqldb, err := sql.Open("nrmysql", r.dbConfig.DbURI)
		if err != nil {
			return fmt.Errorf("failed to open newrelic mysql connection: %w", err)
		}
		gdb, err = gorm.Open(mysql.New(mysql.Config{Conn: sqldb}), &gorm.Config{})
		if err != nil {
			return fmt.Errorf("failed to open DB connection with newrelic instrumentation: %w", err)
		}
	} else {
		gdb, err = gorm.Open(mysql.Open(r.dbConfig.DbURI), &gorm.Config{})
		if err != nil {
			return fmt.Errorf("failed to open DB connection: %w", err)
		}
	}

	r.DB = gdb

	db, err := gdb.DB()
	if err != nil {
		return fmt.Errorf("failed to retrieve underlying sqldb object: %w", err)
	}
	db.SetMaxOpenConns(r.dbConfig.DbMaxDBConnections)

	return nil
}

// SetNewRelicApp enables New Relic instrumentation on this repository. If already initialized, it reconnects using the NR driver.
func (r *Repository) SetNewRelicApp(app *newrelic.Application) error {
	r.nrApp = app
	if r.DB == nil {
		return nil
	}
	sqldb, err := r.DB.DB()
	if err == nil {
		sqldb.Close()
	}
	r.DB = nil
	return r.init()
}

// RunMigrations migrates all provided entities which must be gorm compatible entities
func (r *Repository) RunMigrations(entities []interface{}) {
	for _, e := range entities {
		r.DB.AutoMigrate(e)
	}
}
