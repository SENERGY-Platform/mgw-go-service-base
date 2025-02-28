/*
 * Copyright 2024 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sql_db_hdl

import (
	"bufio"
	"context"
	"database/sql"
	"io"
	"os"
	"strings"
	"time"
)

func InitDB(ctx context.Context, db *sql.DB, schemaPath string, delay, timeout time.Duration, migrations ...Migration) error {
	err := waitForDB(ctx, db, delay)
	if err != nil {
		return err
	}
	file, err := os.Open(schemaPath)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	var stmts []string
	for {
		stmt, err := reader.ReadString(';')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		stmts = append(stmts, strings.TrimSuffix(stmt, ";"))
	}
	ctxWt, cf := context.WithTimeout(ctx, timeout)
	defer cf()
	for _, stmt := range stmts {
		_, err = db.ExecContext(ctxWt, stmt)
		if err != nil {
			return err
		}
	}
	for _, migration := range migrations {
		ok, err := migration.Required(ctx, db, timeout)
		if err != nil {
			return err
		}
		if ok {
			if err = migration.Run(ctx, db, timeout); err != nil {
				return err
			}
		}
	}
	return nil
}

func waitForDB(ctx context.Context, db *sql.DB, delay time.Duration) error {
	err := db.PingContext(ctx)
	if err == nil {
		return nil
	} else {
		if Logger != nil {
			Logger.Error(err)
		}
	}
	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err = db.PingContext(ctx)
			if err == nil {
				return nil
			} else {
				if Logger != nil {
					Logger.Error(err)
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
