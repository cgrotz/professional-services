// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package data_access

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

var Db *sql.DB

func InitDatabase() {
	var err error
	cfg := mysql.Config{
		User:                 os.Getenv("DATABASE_USER"),
		Passwd:               os.Getenv("DATABASE_PASSWORD"),
		Net:                  os.Getenv("DATABASE_NET"),
		Addr:                 os.Getenv("DATABASE_HOST"),
		DBName:               os.Getenv("DATABASE_NAME"),
		MultiStatements:      true,
		AllowNativePasswords: true,
	}
	// Get a database handle.
	Db, err = sql.Open("mysql", cfg.FormatDSN())
	Db.SetMaxOpenConns(100)
	Db.SetMaxIdleConns(5)
	if err != nil {
		log.Fatal(err)
	}
	err = MigrateDatabase(os.Getenv("DATABASE_NAME"), Db)
	if err != nil {
		log.Fatal("Unable to initalize database")
	}
}

func Close() {
	Db.Close()
}

func GetTransaction() (*sql.Tx, error) {
	ctx := context.Background()
	return Db.BeginTx(ctx, nil)
}
