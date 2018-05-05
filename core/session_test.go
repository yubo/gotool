package core

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"golang.org/x/net/context"
)

func mustExec(t *testing.T, db *Db, query string, args ...interface{}) (res sql.Result) {
	res, err := db.Exec(query, args...)
	if err != nil {
		if len(query) > 300 {
			query = "[query too large to print]"
		}
		t.Fatalf("error on %s: %s", query, err.Error())
	}
	return res
}

func TestSession(t *testing.T) {
	var (
		sess  *Session
		err   error
		store *SessionStore
		sid   string
	)

	if !available {
		t.Skipf("MySQL server not running on %s", netAddr)
	}

	cf := SessionConfig{
		CookieName:     "test_sid",
		SidLength:      24,
		HttpOnly:       true,
		Domain:         "",
		Dsn:            dsn,
		GcLifetime:     60,
		MaxLifetime:    3600,
		CookieLifetime: 86400,
	}

	ctx, cancel := context.WithCancel(context.Background())
	if sess, err = NewSession(cf, ctx); err != nil {
		t.Fatalf("error NewSession: %s", err.Error())
	}
	defer cancel()

	mustExec(t, sess.db, "DROP TABLE IF EXISTS session;")
	mustExec(t, sess.db, "CREATE TABLE `session` ( `session_key` char(64) NOT NULL, `session_data` blob, `last_time` int(11) unsigned NOT NULL, PRIMARY KEY (`session_key`)) ENGINE=MyISAM DEFAULT CHARSET=utf8; ")
	defer sess.db.Exec("DROP TABLE IF EXISTS session")

	r, _ := http.NewRequest("GET", "", bytes.NewBuffer([]byte{}))
	w := httptest.NewRecorder()

	if store, err = sess.Start(w, r); err != nil {
		t.Fatalf("session.Start(): %s", err.Error())
	}

	if n := sess.All(); n != 1 {
		t.Fatalf("sess.All() got %d want %d", n, 1)
	}

	store.Set("abc", "11223344")
	if err = store.Update(w); err != nil {
		t.Fatalf("store.Update(w) got err %s ", err.Error())
	}
	sid = store.Sid()

	// new request
	r, _ = http.NewRequest("GET", "", bytes.NewBuffer([]byte{}))
	w = httptest.NewRecorder()

	cookie := &http.Cookie{
		Name:     cf.CookieName,
		Value:    url.QueryEscape(sid),
		Path:     "/",
		HttpOnly: cf.HttpOnly,
		Domain:   cf.Domain,
	}
	if cf.CookieLifetime > 0 {
		cookie.MaxAge = cf.CookieLifetime
		cookie.Expires = time.Now().Add(time.Duration(cf.CookieLifetime) * time.Second)
	}
	http.SetCookie(w, cookie)
	r.AddCookie(cookie)
	if store, err = sess.Start(w, r); err != nil {
		t.Fatalf("session.Start(): %s", err.Error())
	}

	if n := sess.All(); n != 1 {
		t.Fatalf("sess.All() got %d want %d", n, 1)
	}

	if v, ok := store.Get("abc").(string); !(ok && v == "11223344") {
		t.Fatalf("store.Get('abc') got %s want %s", v, "11223344")
	}

	store.Set("abc", "22334455")

	if v, ok := store.Get("abc").(string); !(ok && v == "22334455") {
		t.Fatalf("store.Get('abc') got %s want %s", v, "22334455")
	}

	sess.Destroy(w, r)
	if n := sess.All(); n != 0 {
		t.Fatalf("sess.All() got %d want %d", n, 0)
	}

}

func TestSessionGC(t *testing.T) {
	var (
		sess *Session
		err  error
	)

	if !available {
		t.Skipf("MySQL server not running on %s", netAddr)
	}

	cf := SessionConfig{
		CookieName:     "test_sid",
		SidLength:      24,
		HttpOnly:       true,
		Domain:         "",
		Dsn:            dsn,
		GcLifetime:     1,
		MaxLifetime:    1,
		CookieLifetime: 86400,
	}

	ctx, cancel := context.WithCancel(context.Background())
	if sess, err = NewSession(cf, ctx); err != nil {
		t.Fatalf("error NewSession: %s", err.Error())
	}
	defer cancel()

	mustExec(t, sess.db, "DROP TABLE IF EXISTS session;")
	mustExec(t, sess.db, "CREATE TABLE `session` ( `session_key` char(64) NOT NULL, `session_data` blob, `last_time` int(11) unsigned NOT NULL, PRIMARY KEY (`session_key`)) ENGINE=MyISAM DEFAULT CHARSET=utf8; ")
	//defer sess.db.Exec("DROP TABLE IF EXISTS session")

	r, _ := http.NewRequest("GET", "", bytes.NewBuffer([]byte{}))
	w := httptest.NewRecorder()

	if _, err = sess.Start(w, r); err != nil {
		t.Fatalf("session.Start(): %s", err.Error())
	}
	if n := sess.All(); n != 1 {
		t.Fatalf("sess.All() got %d want %d", n, 1)
	}

	time.Sleep(time.Millisecond * 3000)
	if n := sess.All(); n != 0 {
		t.Fatalf("sess.All() got %d want %d", n, 0)
	}

}
