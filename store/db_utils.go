package store

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

var (
	DRIVER_MYSQL = "mysql"
	DNS_MYSQL    = "root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4"
	//DNS_MYSQL = "root:123456@tcp(149.28.68.93:3306)/lemochain01?charset=utf8mb4"
)

var (
	tables = []string{
		`CREATE TABLE IF NOT EXISTS t_kv (
  				lm_flg int(11) NOT NULL,
  				lm_key varchar(256) NOT NULL,
  				lm_val blob NOT NULL,
  				lm_pos bigint(22) NOT NULL,
  				st timestamp NOT NULL 
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
	}
)

// driver = "mysql"
// dns = root:123123@tcp(localhost:3306)/lemochain?charset=utf8mb4
func Open(driver string, dns string) (*sql.DB, error) {
	db, err := sql.Open(driver, dns)
	if err != nil {
		return nil, err
	} else {
		InitTable(db)
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
	_, err := db.Exec("REPLACE INTO t_kv(lm_key, lm_val) VALUES (?,?)", key, val)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func TxSet(db *sql.DB, hash, from, to string, val []byte, ver int64, st int64) error {
	_, err := db.Exec("REPLACE INTO t_tx(tx_key, tx_from, tx_to, tx_val, tx_ver, tx_st) VALUES (?,?,?,?,?,?)", hash, from, to, val, ver, st)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func TxGet8Hash(db *sql.DB, hash string) ([]byte, int64, error) {
	row := db.QueryRow("SELECT tx_val, tx_st FROM t_tx WHERE tx_key = ?", hash)
	var val []byte
	var st int64
	err := row.Scan(&val, &st)
	if err != nil {
		return nil, -1, err
	} else {
		return val, st, nil
	}
}

func TxGet8AddrNext(db *sql.DB, addr string, start int64, size int) ([][]byte, []int64, int64, error) {
	stmt, err := db.Prepare("SELECT tx_val, tx_ver, tx_st FROM t_tx WHERE (tx_from = ? or tx_to = ?) and (tx_ver > ?) ORDER BY tx_ver ASC LIMIT 0, ?")
	if err != nil {
		return nil, nil, -1, err
	}

	rows, err := stmt.Query(addr, addr, start, size)
	if err != nil {
		return nil, nil, -1, err
	}

	resultVal := make([][]byte, 0)
	resultSt := make([]int64, 0)
	maxVer := start
	for rows.Next() {
		var val []byte
		var ver int64
		var st int64
		err := rows.Scan(&val, &ver, &st)
		if err != nil {
			return nil, nil, -1, err
		}

		resultVal = append(resultVal, val)
		resultSt = append(resultSt, st)
		if maxVer < ver {
			maxVer = ver
		}
	}
	return resultVal, resultSt, maxVer, nil
}

func TxGet8AddrPre(db *sql.DB, addr string, start int64, size int) ([][]byte, []int64, int64, error) {
	stmt, err := db.Prepare("SELECT tx_val, tx_ver, tx_st FROM t_tx WHERE (tx_from = ? or tx_to = ?) and (tx_ver < ?) ORDER BY tx_ver DESC LIMIT 0, ?")
	if err != nil {
		return nil, nil, -1, err
	}

	rows, err := stmt.Query(addr, addr, start, size)
	if err != nil {
		return nil, nil, -1, err
	}

	resultVal := make([][]byte, 0)
	resultSt := make([]int64, 0)
	maxVer := start
	for rows.Next() {
		var val []byte
		var ver int64
		var st int64
		err := rows.Scan(&val, &ver, &st)
		if err != nil {
			return nil, nil, -1, err
		}

		resultVal = append(resultVal, val)
		resultSt = append(resultSt, st)
		if maxVer > ver {
			maxVer = ver
		}
	}
	return resultVal, resultSt, maxVer, nil
}

func Del(db *sql.DB, key string) error {
	_, err := db.Exec("DELETE FROM t_kv WHERE lm_key = ?", key)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func InitTable(db *sql.DB) {
	for _, tb := range tables {
		_, err := db.Exec(tb)
		if err != nil {
			panic(fmt.Sprintf("init mysql's table error: %v", err))
		}
	}
}
