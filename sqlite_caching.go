package main

import (
	"os"
	"sync"
	"gosqlite.googlecode.com/hg/sqlite"
)

var CacheSqliteConnections bool = true
var CachedSqliteConnections map[string]*sqlite.Conn = make(map[string]*sqlite.Conn)
var connectionCount int = 0
var connectionCountLimit int = 3
var CachedSqliteMutex *sync.Mutex = new(sync.Mutex)

func SqliteCachedOpen(filename string) (*sqlite.Conn, os.Error) {
	if !CacheSqliteConnections {
		return sqlite.Open(filename)
	}
	
	CachedSqliteMutex.Lock()
	defer CachedSqliteMutex.Unlock()

	if conn := CachedSqliteConnections[filename]; conn != nil {
		Logf(DEBUG, "Using cached connection\n")
		return conn, nil
	}

	conn, err := sqlite.Open(filename)
	CachedSqliteConnections[filename] = conn
	connectionCount++
	return conn, err
}

func SqliteCachedClose(conn *sqlite.Conn) {
	if CacheSqliteConnections {
		if connectionCount < connectionCountLimit {
			Log(DEBUG, "Not closing connection, we are caching them")
			return
		}

		Log(ERROR, "There are too many open connections")
		return
	}
	
	Log(DEBUG, "Closing connection")
	must(conn.Close())
}