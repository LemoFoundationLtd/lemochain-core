package store

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

var (
	DRIVER_MYSQL = "mysql"
	//DNS_MYSQL    = "root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4"
	DNS_MYSQL = "root:123456@tcp(149.28.68.93:3306)/lemochain01?charset=utf8mb4"
)

// driver = "mysql"
// dns = root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4
func Open(driver string, dns string) (*sql.DB, error) {
	db, err := sql.Open(driver, dns)
	if err != nil {
		return nil, err
	} else {
		return db, nil
	}
}

func clear(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM t_kv")
	if err != nil {
		return err
	} else {
		return nil
	}
}

func Get(db *sql.DB, key string) ([]byte, error) {
	row := db.QueryRow("SELECT lm_val FROM t_kv WHERE lm_key = ?", key)
	var val []byte
	err := row.Scan(&val)
	if err != nil {
		return nil, err
	} else {
		return val, nil
	}
}

func Set(db *sql.DB, key string, val []byte) error {
	_, err := db.Exec("INSERT INTO t_kv(lm_key, lm_val) VALUES (?,?)", key, val)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func Del(db *sql.DB, key string) error {
	_, err := db.Exec("DELETE FROM t_kv WHERE lm_key = ?", key)
	if err != nil {
		return err
	} else {
		return nil
	}
}
