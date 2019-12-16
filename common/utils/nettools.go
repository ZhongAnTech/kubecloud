package utils

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

var (
	HttpClient = &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).Dial,
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 200,
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 60 * time.Second,
	}
)

func TcpConnTest(server string) error {
	conn, err := net.DialTimeout("tcp", server, time.Second*3)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}
