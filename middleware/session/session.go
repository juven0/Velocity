package session

import (
	"encoding/base64"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/juven0/Velocity/types"
	"github.com/valyala/fasthttp"
)

type Session struct {
	ID        string
	Data      map[string]interface{}
	CreatedAt time.Time
	UpdateAt  time.Time
	ExpiredAt time.Time
}

type MemoryStore struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
}

type SesionConfig struct {
	Store       MemoryStore
	CookiName   string
	CookiPath   string
	cookiDomain string
	MaxAge      time.Duration
	Secure      bool
	HTTPOnly    bool
	SameSite    fasthttp.CookieSameSite
	GCInterval  time.Duration
}

func generateSID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func (ms *MemoryStore) Get(id string) (*Session, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	session, exist := ms.sessions[id]
	if !exist {
		return nil, errors.New("ErrSessionNotFound")
	}

	if time.Now().After(session.ExpiredAt) {
		return nil, errors.New("ErrSessionExpired")
	}

	return session, nil
}

func (ms *MemoryStore) Set(session *Session) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	session.UpdateAt = time.Now()
	ms.sessions[session.ID] = session
	return nil
}

func (ms *MemoryStore) Delete(sessionID string) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	delete(ms.sessions, sessionID)

	return nil
}

func (s *Session) CreateSession(userID string, store *MemoryStore) (*Session, error) {
	session := &Session{
		ID:        generateSID(),
		Data:      make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdateAt:  time.Now(),
		ExpiredAt: time.Now().Add(24 * time.Hour),
	}
	err := store.Set(session)
	return session, err
}

func GetSession(ctx *types.Context, store *MemoryStore, cookiName string) (*Session, error) {
	sessionID := string(ctx.Request.Header.Cookie(cookiName))
	if sessionID == "" {
		return nil, errors.New("ErrNoSessionID")
	}
	session, err := store.Get(sessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiredAt) {
		store.Delete(sessionID)
		return nil, errors.New("ErrSessionExpired")
	}

	return session, nil
}

func UpdateSession(session *Session, store *MemoryStore) error {
	session.UpdateAt = time.Now()

	session.ExpiredAt = time.Now().Add(24 * time.Hour)

	return store.Set(session)
}

func DestroySession(ctx *types.Context, store *MemoryStore, cookiName string) error {
	sessionID := string(ctx.Request.Header.Cookie(cookiName))
	if sessionID == "" {
		return nil
	}

	err := store.Delete(sessionID)

	cookie := &fasthttp.Cookie{}
	cookie.SetKey(cookiName)
	cookie.SetValue("")
	cookie.SetMaxAge(-1)
	cookie.SetPath("/")
	ctx.Response.Header.SetCookie(cookie)

	return err
}

func SessionMiddleware(config *SesionConfig) types.HandlerFunc {
	return func(ctx *types.Context) error {
		session, err := GetSession(ctx, &config.Store, config.CookiName)

		if err != nil || session == nil {
			session = &Session{
				ID:        generateSID(),
				Data:      make(map[string]interface{}),
				CreatedAt: time.Now(),
				UpdateAt:  time.Now(),
				ExpiredAt: time.Now().Add(config.MaxAge),
			}

			config.Store.Set(session)

			cookie := &fasthttp.Cookie{}
			cookie.SetKey(config.CookiName)
			cookie.SetValue(session.ID)
			cookie.SetMaxAge(int(config.MaxAge))
			cookie.SetPath(config.CookiPath)
			cookie.SetDomain(config.cookiDomain)
			cookie.SetHTTPOnly(config.HTTPOnly)
			cookie.SetSecure(config.Secure)
			cookie.SetSameSite(config.SameSite)

			ctx.Response.Header.SetCookie(cookie)

		}

		ctx.SetUserValue("session", session)

		ctx.Next()

		defer func() {
			if session.UpdateAt.Before(time.Now().Add(-1 * time.Minute)) {
				session.UpdateAt = time.Now()
				config.Store.Set(session)
			}
		}()

		return err
	}
}
