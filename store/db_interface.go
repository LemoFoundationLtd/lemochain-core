package store

import (
	"database/sql"
	"github.com/LemoFoundationLtd/lemochain-go/common"
)

type DBWriter interface {
	SetIndex(flg int, route []byte, key []byte, offset int64) error
}

type DBReader interface {
	GetIndex(key []byte) (int, []byte, int64, error)
}

type DB interface {
	DBWriter
	DBReader
}

type MySqlDB struct {
	engine *sql.DB
	driver string
	dns    string
}

func NewMySqlDB(driver string, dns string) *MySqlDB {
	db, err := Open(driver, dns)
	if err != nil {
		panic("OPEN MYSQL DATABASE ERROR.")
	} else {
		return &MySqlDB{
			engine: db,
			driver: driver,
			dns:    dns,
		}
	}
}

func (db *MySqlDB) SetIndex(flg int, route []byte, key []byte, offset int64) error {
	tmp := common.ToHex(key[:])
	_, err := db.engine.Exec("REPLACE INTO t_kv(lm_flg, lm_key, lm_val, lm_pos) VALUES (?,?,?,?)", flg, tmp, route, offset)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (db *MySqlDB) GetIndex(key []byte) (int, []byte, int64, error) {
	tmp := common.ToHex(key[:])
	row := db.engine.QueryRow("SELECT lm_flg, lm_val, lm_pos FROM t_kv WHERE lm_key = ?", tmp)
	var flg int
	var val []byte
	var pos int64
	err := row.Scan(&flg, &val, &pos)
	if err == sql.ErrNoRows {
		return -1, nil, -1, nil
	}
	if err != nil {
		return -1, nil, -1, err
	}

	return flg, val, pos, nil
}

func (db *MySqlDB) Clear() error {
	_, err := db.engine.Exec("DELETE FROM t_kv")
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (db *MySqlDB) Close() {
	db.engine.Close()
}
