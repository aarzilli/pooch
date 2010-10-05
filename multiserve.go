package main

import (
	"http"
	"fmt"
	"strings"
	"gosqlite.googlecode.com/hg/sqlite"
	"path"
	"crypto/sha512"
)

type MultiuserDb struct {
	conn *sqlite.Conn
	directory string
}

func OpenMultiuserDb(directory string) *MultiuserDb{
	multiuserDb, err := sqlite.Open(path.Join(directory, "users.db"))
	if err != nil { panic(err) }
	MustExec(multiuserDb, "CREATE TABLE for multiuser db", "CREATE TABLE IF NOT EXISTS users (username TEXT, salt TEXT, passhash BLOB)")
	MustExec(multiuserDb, "CREATE TABLE for multiuser db (cookies)", "CREATE TABLE IF NOT EXISTS cookies (username TEXT, cookie TEXT)")
	return &MultiuserDb{multiuserDb, directory}
}

func (mdb *MultiuserDb) SaveIdCookie(username, idCookie string) {
	MustExec(mdb.conn, "INSERT for login", "INSERT INTO cookies(username, cookie) VALUES(?,?)", username, idCookie)
}

func (mdb *MultiuserDb) Exists(username string) bool {
	stmt, err := mdb.conn.Prepare("SELECT username FROM users WHERE username = ?")
	if err != nil { panic(fmt.Sprintf("Error preparing statement for Exists: %s", err.String())) }
	defer stmt.Finalize()
	
	err = stmt.Exec(username)
	if err != nil { panic(fmt.Sprintf("Error executing statement for Exists: %s", err.String())) }
	
	return stmt.Next()
}

func PasswordHashing(salt, password string) []byte {
	hasher := sha512.New()
	hasher.Write(([]uint8)(salt + password))
	hashedPassword := hasher.Sum()
	return hashedPassword
}

func (mdb *MultiuserDb) Verify(username, password string) bool {
	Logf(DEBUG, "Verifying %s / %s\n", username, password)

	stmt, err := mdb.conn.Prepare("SELECT salt, passhash FROM users WHERE username = ?")
	if err != nil { panic(fmt.Sprintf("Error preparing statement for Verify: %s", err)) }
	defer stmt.Finalize()
	
	err = stmt.Exec(username)
	if err != nil { panic(fmt.Sprintf("Error executing statement for Verify: %s", err)) }
	
	if !stmt.Next() { return false }

	var salt string
	var passhash []byte
	scanerr := stmt.Scan(&salt, &passhash)
	if scanerr != nil { panic(fmt.Sprintf("Error reading from the users database: %s", scanerr)) }

	hashedPassword := PasswordHashing(salt, password)
	
	Logf(DEBUG, "Salt for %s is %s, passhash: %v, password to identify is hashed to: %v\n", username, salt, passhash, hashedPassword)

	if len(hashedPassword) != len(passhash) { return false }

	for i, _ := range passhash {
		if hashedPassword[i] != passhash[i] { return false }
	}

	return true
}

func (mdb *MultiuserDb) Register(username, password string) {
	salt := MakeRandomString(8)
	hashedPassword := PasswordHashing(salt, password)
	MustExec(mdb.conn, "INSERT for Register", "INSERT INTO users(username, salt, passhash) VALUES(?, ?, ?)", username, salt, hashedPassword)
}

func (mdb *MultiuserDb) OpenOrCreateUserDb(username string) *Tasklist {
	//TODO: open or create user db
}

func (mdb *MultiuserDb) Close() {
	mdb.conn.Close()
}

var multiuserDb *MultiuserDb

func AddCookies(c http.ResponseWriter, cookies map[string]string) {
	for k, v := range cookies {
		c.SetHeader("Set-Cookie", fmt.Sprintf("%s=%s; Max-Age=65535; path=/", k, v))
	}
}

func GetCookies(c *http.Request) map[string]string {
	cookies := c.Header["Cookie"]
	cookiev := strings.Split(cookies, "=", 2)
	
	r := make(map[string]string)
	if len(cookiev) > 1 {
		r[cookiev[0]] = cookiev[1]
	}
	
	return r
}

func AddIdCookie(c http.ResponseWriter) string {
	cookies := make(map[string]string)
	cookies["id"] = MakeRandomString(20);
	AddCookies(c, cookies);
	return cookies["id"]
}

func GetIdCookie(c *http.Request) string {
	cookies := GetCookies(c)
	return cookies["id"]
}

func LoginServer(c http.ResponseWriter, req *http.Request) {
		defer func() {
		if r := recover(); r != nil {
			LoginHTML(map[string]string{ "problem": fmt.Sprintf("Login failed with internal error: %s\n", r)}, c)
			panic(r)
		}
	}()
	
	if req.FormValue("user") == "" {
		LoginHTML(map[string]string{ "problem": "" }, c)
	} else {
		if !multiuserDb.Verify(req.FormValue("user"), req.FormValue("password")) {
			LoginHTML(map[string]string{ "problem": "No match for " + req.FormValue("user") + " and given password" }, c)
		} else {
			idCookie := AddIdCookie(c)
			multiuserDb.SaveIdCookie(req.FormValue("user"), idCookie)
			LoginOKHTML(nil, c)
		}
	}
}

func RegisterServer(c http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			RegisterHTML(map[string]string{ "problem": fmt.Sprintf("Registration failed with internal error: %s\n", r)}, c)
			panic(r)
		}
	}()
	
	if req.FormValue("user") == "" {
		RegisterHTML(map[string]string{ "problem": "" }, c)
	} else {
		if multiuserDb.Exists(req.FormValue("user")) {
			RegisterHTML(map[string]string{ "problem": "Username " + req.FormValue("user") + " already exists" }, c)
		} else {
			multiuserDb.Register(req.FormValue("user"), req.FormValue("password"))
			RegisterOKHTML(nil, c)
		}
	}
}

func MultiServe(port string, directory string) {
	multiuserDb = OpenMultiuserDb(directory)
	
	http.HandleFunc("/login", WrapperServer(LoginServer))
	http.HandleFunc("/register", WrapperServer(RegisterServer))

	if err := http.ListenAndServe(":" + port, nil); err != nil {
		Log(ERROR, "Couldn't serve: ", err)
		return
	}
}