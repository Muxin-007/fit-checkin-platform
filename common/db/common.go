package db

import (
	"database/sql"
	"fmt"
	"strings"

	"platform/common/config"
)

var (
	defaultCharset   = "utf8mb4"
	defaultCollation = "utf8mb4_general_ci"
)

type Options func()

func WithCharset(charset string) Options {
	return func() {
		defaultCharset = charset
	}
}

func WithCollation(collation string) Options {
	return func() {
		defaultCollation = collation
	}
}

// CreateDatabase creates database for mysql / postgres / sqlite
func CreateDatabase(dbConfig config.DbConfig, opts ...Options) error {
	for _, opt := range opts {
		opt()
	}

	dsn, err := dbConfig.GetDsnWithoutDatabase()
	if err != nil {
		return err
	}
	switch dbConfig.Driver {
	case config.DbDriverMySql:
		return createMySQLDatabase(dsn, dbConfig.ConnCfg.Database)

	case config.DbDriverPostgres:
		return createPostgresDatabase(dsn, dbConfig.ConnCfg.Database, dbConfig.ConnCfg.Schema)

	case config.DbDriverSqlite:
		// sqlite does not support create database
		return nil

	default:
		return fmt.Errorf("unsupported driver: %s", dbConfig.Driver)
	}
}

// createMySQLDatabase creates database for mysql
func createMySQLDatabase(dsn, dbName string) error {
	db, err := sql.Open(string(config.DbDriverMySql), dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		return err
	}

	sql := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s COLLATE %s",
		dbName, defaultCharset, defaultCollation,
	)

	_, err = db.Exec(sql)

	return err
}

// createPostgresDatabase creates database for postgres
func createPostgresDatabase(dsn, dbName, schema string) error {
	// connect to postgres
	db, err := sql.Open(string(config.DbDriverPostgres), dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		return err
	}

	// check if database exists
	var exists bool
	checkSQL := `
		SELECT EXISTS (
			SELECT 1 FROM pg_database WHERE datname = $1
		)`
	if err = db.QueryRow(checkSQL, dbName).Scan(&exists); err != nil {
		return err
	}

	// create database if not exists
	if !exists {
		createDBSQL := fmt.Sprintf(
			`CREATE DATABASE "%s" ENCODING 'UTF8'`,
			dbName,
		)
		if _, err := db.Exec(createDBSQL); err != nil {
			return err
		}
	}

	// create schema
	if schema != "" {
		// connect to target database (IMPORTANT)
		targetDSN, err := replacePostgresDBName(dsn, dbName)
		if err != nil {
			return err
		}

		targetDB, err := sql.Open(string(config.DbDriverPostgres), targetDSN)
		if err != nil {
			return err
		}
		defer targetDB.Close()

		if err := targetDB.Ping(); err != nil {
			return err
		}

		createSchemaSQL := fmt.Sprintf(
			`CREATE SCHEMA IF NOT EXISTS "%s"`,
			schema,
		)
		if _, err := targetDB.Exec(createSchemaSQL); err != nil {
			return err
		}

		// set search_path (strongly recommended)
		setSearchPathSQL := fmt.Sprintf(
			`ALTER DATABASE "%s" SET search_path TO "%s", public`,
			dbName,
			schema,
		)
		if _, err := targetDB.Exec(setSearchPathSQL); err != nil {
			return err
		}
	}

	return nil
}

func replacePostgresDBName(dsn, dbName string) (string, error) {
	parts := strings.Fields(dsn)

	found := false
	for i, part := range parts {
		if strings.HasPrefix(part, "dbname=") {
			parts[i] = "dbname=" + dbName
			found = true
			break
		}
	}

	if !found {
		parts = append(parts, "dbname="+dbName)
	}

	return strings.Join(parts, " "), nil
}
