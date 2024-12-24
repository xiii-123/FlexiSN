package db

import (
	"database/sql"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"sync"
	"time"
)

// DBManager 管理数据库的结构体
type DBManager struct {
	db        *sql.DB
	filePath  string
	memoryDB  *sql.DB
	lock      sync.RWMutex
	closeChan chan struct{}
}

// OpenDB 打开数据库（支持内存数据库或持久化数据库文件）
func OpenDB(dbFile string) (*sql.DB, error) {
	var db *sql.DB
	var err error
	if dbFile == "" {
		// 使用内存数据库
		db, err = sql.Open("sqlite3", ":memory:")
	} else {
		// 使用持久化数据库文件
		db, err = sql.Open("sqlite3", dbFile)
	}
	if err != nil {
		return nil, err
	}

	// 创建表，如果表不存在的话
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS kv_store (
		key TEXT PRIMARY KEY,
		value TEXT
	)`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// NewDBManager 创建一个新的 DBManager，初始化内存数据库并加载持久化数据（如果有）
func NewDBManager(dbFile string) (*DBManager, error) {
	// 如果文件路径不为空，尝试打开持久化数据库
	db, err := OpenDB(dbFile)
	if err != nil {
		// 如果数据库文件不存在（错误类型为 "sqlite3: database disk image is malformed"），
		// 创建新数据库文件
		if err.Error() == "sqlite3: database disk image is malformed" {
			// 如果数据库文件损坏，直接清空并重建
			os.Remove(dbFile)
			db, err = OpenDB(dbFile)
			if err != nil {
				return nil, err
			}
		} else {
			// 如果其他错误（如文件不存在），尝试创建新的数据库
			db, err = OpenDB(dbFile)
			if err != nil {
				return nil, err
			}
		}
	}

	// 创建内存数据库
	memoryDB, err := OpenDB("")
	if err != nil {
		return nil, err
	}

	// 如果数据库文件存在，则加载数据到内存数据库
	if dbFile != "" {
		err = LoadFromDB(db, memoryDB)
		if err != nil {
			return nil, err
		}
	}

	manager := &DBManager{
		db:        db,
		filePath:  dbFile,
		memoryDB:  memoryDB,
		closeChan: make(chan struct{}),
	}

	return manager, nil
}

// CloseDB 关闭内存数据库和持久化数据库
func (m *DBManager) CloseDB() {
	close(m.closeChan)
	m.db.Close()
	m.memoryDB.Close()
}

// LoadFromDB 从持久化数据库加载数据到内存数据库
func LoadFromDB(persistentDB, memoryDB *sql.DB) error {
	rows, err := persistentDB.Query("SELECT key, value FROM kv_store")
	if err != nil {
		return err
	}
	defer rows.Close()

	// 将持久化数据库的数据加载到内存数据库
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		_, err := memoryDB.Exec("INSERT OR REPLACE INTO kv_store (key, value) VALUES (?, ?)", key, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// SaveToDB 将内存数据库数据保存到持久化数据库
func (m *DBManager) SaveToDB() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// 清空持久化数据库表
	_, err := m.db.Exec("DELETE FROM kv_store")
	if err != nil {
		return err
	}

	// 从内存数据库复制数据到持久化数据库
	rows, err := m.memoryDB.Query("SELECT key, value FROM kv_store")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return err
		}
		_, err := m.db.Exec("INSERT OR REPLACE INTO kv_store (key, value) VALUES (?, ?)", key, value)
		if err != nil {
			return err
		}
	}
	return nil
}

// PeriodicSave 定期保存内存数据库到持久化数据库
func (m *DBManager) PeriodicSave(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.closeChan:
			return
		case <-ticker.C:
			err := m.SaveToDB()
			if err != nil {
				log.Printf("Error saving to DB: %v", err)
			} else {
				log.Println("Data saved to DB.")
			}
		}
	}
}

// SaveToMemory 将数据存储到内存数据库
func (m *DBManager) SaveToMemory(key string, value interface{}) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// 将结构体序列化为 JSON 字符串
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// 插入数据到内存数据库
	_, err = m.memoryDB.Exec("INSERT OR REPLACE INTO kv_store (key, value) VALUES (?, ?)", key, valueJSON)
	return err
}

// LoadFromMemory 从内存数据库加载数据
func (m *DBManager) LoadFromMemory(key string, result interface{}) error {
	m.lock.RLock()
	defer m.lock.RUnlock()

	var valueJSON string
	err := m.memoryDB.QueryRow("SELECT value FROM kv_store WHERE key = ?", key).Scan(&valueJSON)
	if err != nil {
		return err
	}

	// 将 JSON 字符串反序列化为结构体
	err = json.Unmarshal([]byte(valueJSON), result)
	return err
}
