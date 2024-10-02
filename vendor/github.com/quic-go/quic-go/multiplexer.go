package quic

import (
	"fmt"
	"net"
	"sync"

	"github.com/quic-go/quic-go/internal/utils"
)

var (
	connMuxerOnce sync.Once
	connMuxer     multiplexer
)

type indexableConn interface{ LocalAddr() net.Addr }

type multiplexer interface {
	AddConn(conn indexableConn)
	RemoveConn(indexableConn) error
}

type connMultiplexer struct {
	mutex sync.Mutex

	conns  map[string][]indexableConn // Changed to store a slice of indexableConn
	logger utils.Logger
}

var _ multiplexer = &connMultiplexer{}

func getMultiplexer() multiplexer {
	connMuxerOnce.Do(func() {
		connMuxer = &connMultiplexer{
			conns:  make(map[string][]indexableConn),
			logger: utils.DefaultLogger.WithPrefix("muxer"),
		}
	})
	return connMuxer
}

func (m *connMultiplexer) index(addr net.Addr) string {
	return addr.Network() + " " + addr.String()
}

func (m *connMultiplexer) AddConn(c indexableConn) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	connIndex := m.index(c.LocalAddr())
	m.conns[connIndex] = append(m.conns[connIndex], c)
}

func (m *connMultiplexer) RemoveConn(c indexableConn) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	connIndex := m.index(c.LocalAddr())
	conns, ok := m.conns[connIndex]
	if !ok {
		return fmt.Errorf("cannot remove connection, connection is unknown")
	}

	for i, conn := range conns {
		if conn == c {
			m.conns[connIndex] = append(conns[:i], conns[i+1:]...)
			if len(m.conns[connIndex]) == 0 {
				delete(m.conns, connIndex)
			}
			return nil
		}
	}
	return fmt.Errorf("cannot remove connection, connection not found")
}
