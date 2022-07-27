package raid

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type TCPServer struct {
	port         uint16
	apiKeys      []string
	updaterState *UpdaterState
	updates      *Topic[Update]
}

func NewTCPServer(port uint16, apiKeys []string, updaterState *UpdaterState, updates *Topic[Update]) *TCPServer {
	return &TCPServer{
		port:         port,
		apiKeys:      apiKeys,
		updaterState: updaterState,
		updates:      updates,
	}
}

func (t *TCPServer) Run(ctx context.Context, wg *sync.WaitGroup, errch chan error) {
	defer log.Debug("mapgenerator: exit")

	defer wg.Done()
	wg.Add(1)

	cfg := &net.ListenConfig{
		// Control: func(network string, address string, conn syscall.RawConn) error {
		// 	return conn.Control(func(descriptor uintptr) {
		// 		syscall.SetsockoptInt(int(descriptor), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
		// 	})
		// },
	}

	l, err := cfg.Listen(ctx, "tcp", fmt.Sprintf("0.0.0.0:%d", t.port))
	if err != nil {
		errch <- fmt.Errorf("tcpserver: listen: %w", err)

		return
	}

	defer l.Close()

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				errch <- fmt.Errorf("tcpserver: accept: %w", err)
			}

			return
		}

		log.Debugf("tcpserver: accept %s", conn.RemoteAddr())

		go t.HandleConn(ctx, conn)
	}
}

func (t *TCPServer) HandleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	if err := conn.SetReadDeadline(time.Now().Add(time.Second * 3)); err != nil {
		log.Errorf("tcpserver: set deadline: %v", err)

		return
	}

	buf := make([]byte, 128)

	n, err := conn.Read(buf)
	if err != nil {
		log.Errorf("tcpserver: read auth: %v", err)

		_, _ = conn.Write([]byte("a:timeout\n"))

		return
	}

	parts := strings.Split(strings.TrimSpace(string(buf[:n])), ",")
	apiKey := parts[0]

	id := 0
	if len(parts) > 1 {
		id, _ = strconv.Atoi(parts[1])
	}

	authSuccess := false

	for _, other := range t.apiKeys {
		if other == apiKey {
			authSuccess = true
		}
	}

	log.Debugf("tcpserver: client auth success: %v, key: %v", authSuccess, apiKey)

	if !authSuccess {
		_, _ = conn.Write([]byte("a:wrong_api_key\n"))

		return
	}

	if _, err := conn.Write([]byte("a:ok\n")); err != nil {
		log.Errorf("tcpserver: write auth success: %v", err)

		return
	}

	for _, state := range t.updaterState.States {
		if id == 0 || id == state.ID {
			alert := 0
			if state.Alert {
				alert = 1
			}

			if _, err := conn.Write([]byte(fmt.Sprintf("s:%d=%d\n", state.ID, alert))); err != nil {
				log.Errorf("tcpserver: write state: %v", err)

				return
			}
		}
	}

	events := t.updates.Subscribe("tcpserver-"+conn.RemoteAddr().String(), func(u Update) bool {
		return u.IsFresh && (id == 0 || id == u.State.ID)
	})

	defer func() {
		t.updates.Unsubscribe(events)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}

			alert := 0

			if event.State.Alert {
				alert = 1
			}

			conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
			if _, err := conn.Write([]byte(fmt.Sprintf("s:%d=%d\n", event.State.ID, alert))); err != nil {
				log.Errorf("tcpserver: write state: %v", err)

				return
			}
		case <-time.After(time.Second * 15):
			conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
			if _, err := conn.Write([]byte(fmt.Sprintf("p:%d\n", rand.Intn(10000)))); err != nil {
				log.Errorf("tcpserver: write ping: %v", err)

				return
			}
		}
	}
}
