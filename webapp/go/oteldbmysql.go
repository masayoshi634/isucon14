package main

import (
	_ "github.com/go-sql-driver/mysql"
)

/*
func GetDBNoOtel() (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%v)/%s?charset=utf8mb4&parseTime=true&loc=Local&interpolateParams=true",
		GetEnv("ISUCON_DB_USER", "isucon"),
		GetEnv("ISUCON_DB_PASSWORD", "isucon"),
		GetEnv("ISUCON_DB_HOST", "127.0.0.1"),
		GetEnv("ISUCON_DB_PORT", "3306"),
		GetEnv("ISUCON_DB_NAME", "isuride"),
	)

	tmpDB, err := sql.Open(
		"mysql",
		dsn,
	)
	if err != nil {
		return nil, err
	}

	WaitDB(tmpDB)

	tmpDB.SetMaxOpenConns(50)
	tmpDB.SetConnMaxLifetime(5 * time.Minute)

	return sqlx.NewDb(tmpDB, "mysql"), nil
}

func GetDB() (*sqlx.DB, error) {
	if GetEnv("OTEL_SDK_DISABLED", "false") == "true" {
		return GetDBNoOtel()
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%v)/%s?charset=utf8mb4&parseTime=true&loc=Local&interpolateParams=true",
		GetEnv("ISUCON_DB_USER", "isucon"),
		GetEnv("ISUCON_DB_PASSWORD", "isucon"),
		GetEnv("ISUCON_DB_HOST", "127.0.0.1"),
		GetEnv("ISUCON_DB_PORT", "3306"),
		GetEnv("ISUCON_DB_NAME", "isuride"),
	)

	tmpDB, err := otelsql.Open(
		"mysql",
		dsn,
		otelsql.WithAttributes(
			semconv.DBSystemPostgreSQL,
		),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			Ping:                 false,
			RowsNext:             false,
			DisableErrSkip:       false,
			DisableQuery:         false,
			OmitConnResetSession: true,
			OmitConnPrepare:      true,
			OmitConnQuery:        false,
			OmitRows:             true,
			OmitConnectorConnect: false,
		}),
	)
	if err != nil {
		return nil, err
	}

	WaitDB(tmpDB)

	tmpDB.SetMaxOpenConns(50)
	tmpDB.SetConnMaxLifetime(5 * time.Minute)

	return sqlx.NewDb(tmpDB, "mysql"), nil
}

func WaitDB(db *sql.DB) {
	for {
		err := db.Ping()
		if err == nil {
			break
		}
		log.Println(fmt.Errorf("failed to ping DB on start up. retrying...: %w", err))
		time.Sleep(time.Second * 1)
	}
	log.Println("Succeeded to connect db!")
}
*/
