package mbserver

import (
	"io"
	"log"

	"github.com/goburrow/serial"
)

// ListenRTU starts the Modbus server listening to a serial device.
// For example:  err := s.ListenRTU(&serial.Config{Address: "/dev/ttyUSB0"})
func (s *Server) ListenRTU(serialConfig *serial.Config) (err error) {

	port, err := serial.Open(serialConfig)
	if err != nil {
		log.Fatalf("failed to open %s: %v\n", serialConfig.Address, err)
	}
	s.ports = append(s.ports, port)

	s.portsWG.Add(1)
	go func() {
		defer s.portsWG.Done()
		s.acceptSerialRequests(port)
	}()
	s.portsWG.Wait()
	return err
}

func (s *Server) acceptSerialRequests(port serial.Port) {
SkipFrameError:
	for {
		select {
		case <-s.portsCloseChan:
			return
		default:
		}
		//buffer := make([]byte, 512)

		//bytesRead, err := port.Read(buffer)
		bytesRead, buffer, err := readTimeout(port)
		if err != nil {
			if err != io.EOF {
				log.Printf("serial read error %v\n", err)
			}
			return
		}

		if bytesRead != 0 {

			// Set the length of the packet to the number of read bytes.
			packet := buffer[:bytesRead]
			if len(packet) < 1 {
				log.Printf("packet len error\n")
				continue
			}
			if s.SlaveId != 0 && packet[0] != s.SlaveId {
				continue // slave id not equal
			}
			frame, err := NewRTUFrame(packet)
			if err != nil {
				log.Printf("bad serial frame error %v\n", err)
				//The next line prevents RTU server from exiting when it receives a bad frame. Simply discard the erroneous
				//frame and wait for next frame by jumping back to the beginning of the 'for' loop.
				log.Printf("Keep the RTU server running!!\n")
				continue SkipFrameError
				//return
			}

			request := &Request{port, frame}

			s.requestChan <- request
		}
	}
}

func readTimeout(port serial.Port) (int, []byte, error){
	var data []byte
	lenth := 0
	//retimes := 0
	timeoutTimes := 0
	for {
		datat := make([]byte, 512)
		n, err := port.Read(datat)
		if err != nil{
			if err == serial.ErrTimeout{
				timeoutTimes++
				if timeoutTimes >= 3{
					return lenth,data, nil
				}
			}else{
				return 0, nil, err
			}
		}
		if n > 0{
			timeoutTimes = 0
			lenth+=n
			data = append(data, datat[:n]...)
		}
	}
}