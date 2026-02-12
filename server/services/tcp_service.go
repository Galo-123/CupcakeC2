package services

import (
	"cupcake-server/pkg/globals"
	"encoding/binary"
	"fmt"
	"github.com/hashicorp/yamux"
	"io"
	"log"
	"net"
	"time"
)

// StartTCPListener starts the raw TCP server with Yamux multiplexing
func StartTCPListener(ln *globals.Listener) {
	addr := fmt.Sprintf("%s:%d", ln.BindIP, ln.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("[TCP] Failed to listen on %s: %v", addr, err)
		ln.Status = "Failed"
		return
	}
	ln.TCPServer = listener
	ln.Status = "Running"
	log.Printf("[TCP] Listening on %s (Multiplexing enabled)", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ln.Status == "Stopped" {
				return
			}
			log.Printf("[TCP] Accept error: %v", err)
			continue
		}
		
		config := yamux.DefaultConfig()
		config.EnableKeepAlive = true
		config.KeepAliveInterval = 30 * time.Second
		config.LogOutput = io.Discard

		session, err := yamux.Server(conn, config)
		if err != nil {
			log.Printf("[TCP] Yamux session failed: %v", err)
			conn.Close()
			continue
		}
		log.Printf("[TCP] New Yamux Session established from %s", conn.RemoteAddr())

		go func() {
			stream, err := session.Accept()
			if err != nil {
				log.Printf("[TCP Error] Session accept failed: %v", err)
				session.Close()
				return
			}
			
			log.Printf("[TCP] Accepted new control stream from %s", stream.RemoteAddr())
			ProcessTCPConnection(stream, stream.RemoteAddr().String(), ln, session)
		}()
	}
}

// SendTCPMessage helper (Moved from transport)
func SendTCPMessage(conn net.Conn, data []byte) error {
	length := uint32(len(data))
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, length)
	
	if _, err := conn.Write(header); err != nil { return err }
	if _, err := conn.Write(data); err != nil { return err }
	return nil
}
