package app

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"

	"github.com/LealKevin/terminus/internal/domain"
)

type clientConn struct {
	conn net.Conn
	w    *bufio.Writer
	enc  *json.Encoder

	mu sync.Mutex

	stateMu sync.Mutex
	player  *domain.Player
	dirty   bool
}

type ClientMsg struct {
	PlayerID  string `json:"playerID"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Direction string `json:"direction"`
}

type serverMsg struct {
	Type   string         `json:"type"`
	Msg    string         `json:"msg"`
	World  *domain.World  `json:"world,omitempty"`
	Player *domain.Player `json:"player,omitempty"`
}

func wrapConnection(c net.Conn) *clientConn {
	w := bufio.NewWriter(c)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &clientConn{conn: c, w: w, enc: enc}
}

func (cc *clientConn) sendJson(v any) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if err := cc.enc.Encode(v); err != nil {
		return err
	}
	return cc.w.Flush()
}

type connRegistry struct {
	mu    sync.RWMutex
	conns map[*clientConn]struct{}
}

func newConnRegistry() *connRegistry {
	return &connRegistry{conns: make(map[*clientConn]struct{})}
}

func (r *connRegistry) add(cc *clientConn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conns[cc] = struct{}{}
}

func (r *connRegistry) remove(cc *clientConn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.conns, cc)
}

func (r *connRegistry) snapshot() []*clientConn {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*clientConn, 0, len(r.conns))
	for c := range r.conns {
		out = append(out, c)
	}
	return out
}
