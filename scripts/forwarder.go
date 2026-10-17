package main

import (
	"io"
	"net"
)

func forward(localAddr, remoteAddr string) {
	l, err := net.Listen("tcp", localAddr)
	if err != nil {
		return
	}
	defer l.Close()
	for {
		c, err := l.Accept()
		if err != nil {
			continue
		}
		go func(src net.Conn) {
			defer src.Close()
			dst, err := net.Dial("tcp", remoteAddr)
			if err != nil {
				return
			}
			defer dst.Close()
			go io.Copy(dst, src)
			io.Copy(src, dst)
		}(c)
	}
}

func main() {
	go forward("127.0.0.1:5432", "host.docker.internal:5432")
	forward("127.0.0.1:6379", "host.docker.internal:6379")
}
