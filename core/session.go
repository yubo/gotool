package core

// mysql session support need create table as sql:
//	CREATE TABLE `session` (
//	`session_key` char(64) NOT NULL,
//	`session_data` blob,
//	`last_time` int(11) unsigned NOT NULL,
//	PRIMARY KEY (`session_key`)
//	) ENGINE=MyISAM DEFAULT CHARSET=utf8;
//
import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/golang/glog"

	_ "github.com/go-sql-driver/mysql"
)

const (
	TABLE_NAME = "session"
)

type Session struct {
	config SessionConfig
	db     *Db
}

type SessionConfig struct {
	CookieName     string `json:"cookie_name"`
	SidLength      int    `json:"sid_length"`
	HttpOnly       bool   `json:"http_only"`
	Domain         string `json:"domain"`
	Dsn            string `json:"dsn"`
	GcLifetime     int64  `json:"gc_lifetime"`
	MaxLifetime    int64  `json:"max_lifetime"`
	CookieLifetime int    `json:"cookie_lifetime"`
}

func NewSession(cf SessionConfig, ctx context.Context) (*Session, error) {

	if cf.MaxLifetime == 0 {
		cf.MaxLifetime = cf.GcLifetime
	}

	if cf.SidLength == 0 {
		cf.SidLength = 16
	}

	db, err := DbOpen("mysql", cf.Dsn)
	if err != nil {
		return nil, err
	}

	tick := time.Tick(time.Duration(cf.GcLifetime) * time.Second)
	lifetime := cf.MaxLifetime
	go func() {
		for {
			select {
			case <-ctx.Done():
				db.Close()
				return
			case now := <-tick:
				db.Exec("delete from "+TABLE_NAME+" where last_time < ?", now.Unix()-lifetime)
			}
		}
	}()

	return &Session{config: cf, db: db}, nil
}

func (p *Session) getSid(r *http.Request) (sid string, err error) {
	var cookie *http.Cookie

	cookie, err = r.Cookie(p.config.CookieName)
	if err != nil || cookie.Value == "" {
		return sid, nil
	}

	return url.QueryUnescape(cookie.Value)
}

// SessionStart generate or read the session id from http request.
// if session id exists, return SessionStore with this id.
func (p *Session) Start(w http.ResponseWriter, r *http.Request) (store *SessionStore, err error) {
	var sid string

	if sid, err = p.getSid(r); err != nil {
		glog.V(3).Info(err)
		return
	}

	if sid != "" && p.Exist(sid) {
		return p.Get(sid)
	}

	// Generate a new session
	sid = RandString(p.config.SidLength)

	store, err = p.Get(sid)
	if err != nil {
		return nil, err
	}
	cookie := &http.Cookie{
		Name:     p.config.CookieName,
		Value:    url.QueryEscape(sid),
		Path:     "/",
		HttpOnly: p.config.HttpOnly,
		Domain:   p.config.Domain,
	}
	if p.config.CookieLifetime > 0 {
		cookie.MaxAge = p.config.CookieLifetime
		cookie.Expires = time.Now().Add(time.Duration(p.config.CookieLifetime) * time.Second)
	}
	http.SetCookie(w, cookie)
	r.AddCookie(cookie)
	return
}

func (p *Session) Destroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(p.config.CookieName)
	if err != nil || cookie.Value == "" {
		return
	}

	sid, _ := url.QueryUnescape(cookie.Value)
	p.db.Exec("DELETE FROM "+TABLE_NAME+" where session_key=?", sid)

	cookie = &http.Cookie{Name: p.config.CookieName,
		Path:     "/",
		HttpOnly: p.config.HttpOnly,
		Expires:  time.Now(),
		MaxAge:   -1}

	http.SetCookie(w, cookie)
}

func (p *Session) Get(sid string) (*SessionStore, error) {
	var sessiondata []byte
	var kv map[string]interface{}

	err := p.db.Query("select session_data from "+TABLE_NAME+" where session_key=?", sid).Row(&sessiondata)

	if err == sql.ErrNoRows {
		p.db.Exec("insert into "+TABLE_NAME+"(`session_key`,`session_data`,`last_time`) values(?,?,?)",
			sid, "", time.Now().Unix())
	}

	if len(sessiondata) == 0 {
		kv = make(map[string]interface{})
	} else {
		err = json.Unmarshal(sessiondata, &kv)
		if err != nil {
			return nil, err
		}
	}

	return &SessionStore{db: p.db, sid: sid, values: kv}, nil

}

func (p *Session) Exist(sid string) bool {
	var sessiondata []byte
	err := p.db.Query("select session_data from "+TABLE_NAME+" where session_key=?", sid).Row(&sessiondata)
	return !(err == sql.ErrNoRows)
}

// All count values in mysql session
func (p *Session) All() (ret int) {
	p.db.Query("select count(*) from " + TABLE_NAME).Row(&ret)
	return
}

// SessionStore mysql session store
type SessionStore struct {
	sync.RWMutex
	db     *Db
	sid    string
	values map[string]interface{}
}

// Set value in mysql session.
// it is temp value in map.
func (p *SessionStore) Set(key string, value interface{}) error {
	p.Lock()
	defer p.Unlock()
	p.values[key] = value
	return nil
}

// Get value from mysql session
func (p *SessionStore) Get(key string) interface{} {
	p.RLock()
	defer p.RUnlock()
	if v, ok := p.values[key]; ok {
		return v
	}
	return nil
}

// Delete value in mysql session
func (p *SessionStore) Delete(key string) error {
	p.Lock()
	defer p.Unlock()
	delete(p.values, key)
	return nil
}

// Flush clear all values in mysql session
func (p *SessionStore) Flush() error {
	p.Lock()
	defer p.Unlock()
	p.values = make(map[string]interface{})
	return nil
}

// Sid get session id of this mysql session store
func (p *SessionStore) Sid() string {
	return p.sid
}

func (p *SessionStore) Update(w http.ResponseWriter) error {
	b, err := json.Marshal(p.values)
	if err != nil {
		return err
	}
	p.db.Exec("UPDATE "+TABLE_NAME+" set `session_data`=?, `last_time`=? where session_key=?",
		b, time.Now().Unix(), p.sid)
	return nil
}
