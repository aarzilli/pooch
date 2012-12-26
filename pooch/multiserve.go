/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package pooch

import (
	"net/http"
	"fmt"
	"code.google.com/p/gosqlite/sqlite"
	"path"
	"regexp"
	"crypto/sha512"
)

type MultiuserDb struct {
	conn *sqlite.Conn
	directory string
}

var SecureCookies = true

func OpenMultiuserDb(directory string) *MultiuserDb{
	multiuserDb, err := sqlite.Open(path.Join(directory, "users.db"))
	Must(err)
	MustExec(multiuserDb, "CREATE TABLE IF NOT EXISTS users (username TEXT, salt TEXT, passhash BLOB)")
	MustExec(multiuserDb, "CREATE TABLE IF NOT EXISTS cookies (username TEXT, cookie TEXT)")
	MustExec(multiuserDb, "CREATE TABLE IF NOT EXISTS tokens (username TEXT, token TEXT)")
	return &MultiuserDb{multiuserDb, directory}
}

func (mdb *MultiuserDb) SaveIdCookie(username, idCookie string) {
	MustExec(mdb.conn, "INSERT INTO cookies(username, cookie) VALUES(?,?)", username, idCookie)
}

func (mdb *MultiuserDb) AddAPIToken(username string) {
	token := MakeRandomString(40);
	MustExec(mdb.conn, "INSERT INTO tokens(username, token) VALUES(?,?)", username, token)
}

func (mdb *MultiuserDb) RemoveAPIToken(username, token string) {
	MustExec(mdb.conn, "DELETE FROM tokens WHERE username = ? AND token = ?;", username, token)
}

func (mdb *MultiuserDb) ListAPITokens(username string) []string {
	stmt, serr := mdb.conn.Prepare("SELECT token FROM tokens WHERE username = ?")
	Must(serr)
	defer stmt.Finalize()
	Must(stmt.Exec(username))

	r := make([]string, 0)

	for stmt.Next() {
		var s string
		stmt.Scan(&s)
		r = append(r, s)
	}

	return r
}

func (mdb *MultiuserDb) Exists(username string) bool {
	stmt, err := mdb.conn.Prepare("SELECT username FROM users WHERE username = ?")
	Must(err)
	defer stmt.Finalize()

	Must(stmt.Exec(username))

	return stmt.Next()
}

func PasswordHashing(salt, password string) []byte {
	hasher := sha512.New()
	hasher.Write(([]uint8)(salt + password))
	hashedPassword := hasher.Sum(nil)
	return hashedPassword
}

var InvalidUsernameRE *regexp.Regexp = regexp.MustCompile("[^a-zA-Z0-9]")

func ValidUserName(username string) bool {
	return InvalidUsernameRE.FindIndex(([]byte)(username)) == nil
}

func (mdb *MultiuserDb) Verify(username, password string) bool {
	Logf(DEBUG, "Verifying %s / %s\n", username, password)

	stmt, err := mdb.conn.Prepare("SELECT salt, passhash FROM users WHERE username = ?")
	Must(err)
	defer stmt.Finalize()

	Must(stmt.Exec(username))

	if !stmt.Next() { return false }

	var salt string
	var passhash []byte
	Must(stmt.Scan(&salt, &passhash))

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
	MustExec(mdb.conn, "INSERT INTO users(username, salt, passhash) VALUES(?, ?, ?)", username, salt, hashedPassword)
}

func (mdb *MultiuserDb) UsernameFromCookie(req *http.Request) string {
	stmt, err := mdb.conn.Prepare("SELECT username FROM cookies WHERE cookie = ?")
	Must(err)
	defer stmt.Finalize()

	Must(stmt.Exec(GetIdCookie(req)))

	if !stmt.Next() { return "" }

	var username string
	Must(stmt.Scan(&username))

	return username
}

func (mdb *MultiuserDb) OpenOrCreateUserDb(username string) *Tasklist {
	if username == "" { return nil }
	file := path.Join(mdb.directory, username + ".pooch")
	return OpenOrCreate(file)
}

func (mdb *MultiuserDb) WithOpenUser(req *http.Request, fn func(tl *Tasklist)) bool{
	username := mdb.UsernameFromCookie(req)
	if username != "" {
		tl := mdb.OpenOrCreateUserDb(username)
		defer tl.Close()
		tl.mutex.Lock()
		defer tl.mutex.Unlock()
		fn(tl)
		return true
	}
	return false
}

func MultiWrapperTasklistServer(fn TasklistServer) http.HandlerFunc {
	return func(c http.ResponseWriter, req *http.Request) {
		if !multiuserDb.WithOpenUser(req, func (tl *Tasklist) {
			fn(c, req, tl)
		}) {
			MustLogInHTML(nil, c)
		}
	}
}

func MultiWrapperTasklistWithIdServer(fn TasklistWithIdServer) http.HandlerFunc {
	return func(c http.ResponseWriter, req *http.Request) {
		if !multiuserDb.WithOpenUser(req, func (tl *Tasklist) {
			id := req.FormValue("id")
			if !tl.Exists(id) { panic(fmt.Sprintf("Non-existent id specified")) }
			fn(c, req, tl, id)
		}) {
			MustLogInHTML(nil, c)
		}
	}
}

func (mdb *MultiuserDb) Close() {
	mdb.conn.Close()
}

var multiuserDb *MultiuserDb

func AddCookies(c http.ResponseWriter, cookies map[string]string) {
	for k, v := range cookies {
		s := ""
		if SecureCookies {
			s = " Secure"
		}
		c.Header().Set("Set-Cookie", fmt.Sprintf("%s=%s; Max-Age=2592000; path=/;%s", k, v, s))
		//c.Header().Set("Set-Cookie", fmt.Sprintf("%s=%s; Max-Age=10; path=/", k, v))
	}
}

func GetCookies(c *http.Request) map[string]string {
	r := make(map[string]string)

	for _, cookie := range c.Cookies() {
		r[cookie.Name] = cookie.Value
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
		} else if !ValidUserName(req.FormValue("user")) {
			RegisterHTML(map[string]string{ "problem": "Username " + req.FormValue("user") + " contains invalid characters" }, c)
		} else {
			multiuserDb.Register(req.FormValue("user"), req.FormValue("password"))
			RegisterOKHTML(nil, c)
		}
	}
}

func WhoAmIServer(c http.ResponseWriter, req *http.Request) {
	username := multiuserDb.UsernameFromCookie(req)
	WhoAmIHTML(map[string]string{ "username": username }, c)
}

func APITokensServer(c http.ResponseWriter, req *http.Request) {
	username := multiuserDb.UsernameFromCookie(req)
	switch req.FormValue("action") {
	case "add":
		multiuserDb.AddAPIToken(username)
	case "remove":
		multiuserDb.RemoveAPIToken(username, req.FormValue("token"))
	}
}

func MultiServe(port string, directory string) {
	multiuserDb = OpenMultiuserDb(directory)

	http.HandleFunc("/login", WrapperServer(LoginServer))
	http.HandleFunc("/register", WrapperServer(RegisterServer))
	http.HandleFunc("/whoami", WrapperServer(WhoAmIServer))
	http.HandleFunc("/tokens", WrapperServer(APITokensServer))

	SetupHandleFunc(MultiWrapperTasklistServer, MultiWrapperTasklistWithIdServer, multiuserDb)

	if err := http.ListenAndServe(":" + port, nil); err != nil {
		Log(ERROR, "Couldn't serve: ", err)
		return
	}
}