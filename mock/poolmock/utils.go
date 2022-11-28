package poolmock

import (
	"net"
	"strconv"
	"strings"
)

func parsePort(addr net.Addr) int {
	chunks := strings.Split(addr.String(), ":")
	portString := chunks[len(chunks)-1]
	port, _ := strconv.ParseInt(portString, 10, 64)
	return int(port)
}

func waitCh(chans ...chan any) <-chan any {
	resCh := make(chan any)
	go func() {
		for _, v := range chans {
			<-v
		}
		close(resCh)
	}()
	return resCh
}
