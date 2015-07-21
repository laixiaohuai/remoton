package remoton

import (
	"net"
	"sync"
	"sync/atomic"
)

//Session create from client
type srvSession struct {
	ID        string
	AuthToken string

	mutex   sync.Mutex
	service map[string]chan net.Conn

	Stat struct {
		Services int64
	}
}

func newSession(id, auth string) *srvSession {
	return &srvSession{ID: id,
		AuthToken: auth,
		service:   make(map[string]chan net.Conn),
	}
}

func (c *srvSession) ValidateAuthToken(auth string) bool {
	return c.AuthToken == auth
}

func (c *srvSession) initService(id string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if _, ok := c.service[id]; !ok {
		c.service[id] = make(chan net.Conn)
	}
}

func (c *srvSession) ListenService(id string) <-chan net.Conn {
	c.initService(id)
	return c.service[id]
}

func (c *srvSession) DialService(id string) net.Conn {
	c.initService(id)

	listen, dial := net.Pipe()
	c.service[id] <- listen
	return dial
}

//SessionManager handle sessions
type sessionManager struct {
	sync.Mutex
	sessions map[string]*srvSession

	Stat struct {
		Sessions int64
	}
}

func NewSessionManager() *sessionManager {
	return &sessionManager{sessions: make(map[string]*srvSession)}
}

func (c *sessionManager) Get(id string) *srvSession {
	c.Lock()
	defer c.Unlock()
	return c.sessions[id]
}

func (c *sessionManager) Del(id string) (session *srvSession) {
	c.Lock()
	defer c.Unlock()
	session = c.sessions[id]
	delete(c.sessions, id)

	atomic.AddInt64(&c.Stat.Sessions, -1)
	return
}

func (c *sessionManager) Add(id string, session *srvSession) {
	c.Lock()
	defer c.Unlock()
	atomic.AddInt64(&c.Stat.Sessions, 1)
	c.sessions[id] = session
}
