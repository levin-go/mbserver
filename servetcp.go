package mbserver

import (
	"io"
	"log"
	"net"
	"strings"
)

func (s *Server) accept(listen net.Listener) error {
	for {
		conn, err := listen.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return nil
			}
			log.Printf("Unable to accept connections: %#v\n", err)
			return err
		}
		ips := strings.Split(conn.RemoteAddr().String(), ".")
		if len(ips) < 1 {
			log.Printf("unknown remote address")
			continue
		}
		/*if ips[0] == "10" {
			log.Printf("Unable to accept vpn connections: %s\n", conn.RemoteAddr().String())
			//return errors.New("unable to accept vpn connection")
			continue
		}*/
		go func(conn net.Conn) {
			defer conn.Close()

			for {
				packet := make([]byte, 512)
				bytesRead, err := conn.Read(packet)
				if err != nil {
					if err != io.EOF {
						log.Printf("read error %v\n", err)
					}
					return
				}
				// Set the length of the packet to the number of read bytes.
				packet = packet[:bytesRead]
				if len(packet) < 7 {
					log.Printf("packet len error\n")
					continue
				}
				if s.SlaveId != 0 && packet[6] != s.SlaveId {
					continue // slave id not equal
				}
				frame, err := NewTCPFrame(packet)
				if err != nil {
					log.Printf("bad packet error %v\n", err)
					return
				}

				request := &Request{conn, frame}

				s.requestChan <- request
			}
		}(conn)
	}
}

// ListenTCP starts the Modbus server listening on "address:port".
func (s *Server) ListenTCP(addressPort string) (err error) {
	listen, err := net.Listen("tcp", addressPort)
	if err != nil {
		log.Printf("Failed to Listen: %v\n", err)
		return err
	}
	s.listeners = append(s.listeners, listen)
	go s.accept(listen)
	return err
}
