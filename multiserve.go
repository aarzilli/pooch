package main

import (
	"http"
	"fmt"
	"strings"
	"gosqlite.googlecode.com/hg/sqlite"
	"path"
)

type MultiuserDb struct {
	conn *sqlite.Conn
	directory string
}

func OpenMultiuserDb(directory string) *MultiuserDb{
	multiuserDb, err := sqlite.Open(path.Join(directory, "users.db"))
	if err != nil { panic(err) }
	//TODO: if the db didn't exist create the necessary tables
	return &MultiuserDb{multiuserDb, directory}
}

func (mdb *MultiuserDb) Exists(username string) bool {
	stmt, err := mdb.conn.Prepare("SELECT username FROM users WHERE username = ?")
	if err != nil { panic(fmt.Sprintf("Error preparing statement for Exists: %s", err.String())) }
	defer stmt.Finalize()
	
	err = stmt.Exec(username)
	if err != nil { panic(fmt.Sprintf("Error executing statement for Exists: %s", err.String())) }
	
	return stmt.Next()
}

func (mdb *MultiuserDb) Verify(username, password string) bool {
	stmt, err := mdb.conn.Prepare("SELECT username, salt, passhash FROM users WHERE username = ?")
	if err != nil { panic(fmt.Sprintf("Error preparing statement for Verify: %s", err.String())) }
	defer stmt.Finalize()
	
	err = stmt.Exec(username)
	if err != nil { panic(fmt.Sprintf("Error executing statement for Verify: %s", err.String())) }
	
	if !stmt.Next() { return false }
	
	//TODO: check password
}

func (mdb *MultiuserDb) Register(username, password string) {
	//TODO:
	// - creare salt
	// - creare hash di password + salt
	// - insert statement
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

func AddIdCookie(c http.ResponseWriter) {
	cookies := make(map[string]string)
	cookies["id"] = MakeRandomString(20);
	AddCookies(c, cookies);
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
		if multiuserDb.Verify(req.FormValue("user"), req.FormValue("password")) {
			LoginHTML(map[string]string{ "problem": "No match for " + req.FormValue("user") + " and given password" }, c)
		} else {
			//TODO: settare cookie, confermare riuscita
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