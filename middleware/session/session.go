package session

import (
	"encoding/base64"
	"errors"
	"math/rand"
	"sync"
	"time"

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

func (s *Session) CreateSession(userID string, store MemoryStore) (*Session, error) {
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

func (s *Session) GetSession(ID string) (*Session, error) {}
