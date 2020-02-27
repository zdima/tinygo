package net

import (
	"errors"
	"strings"
)

var ErrInvalidHostPort = errors.New("net: invalid host:port format")

// SplitHostPort splits a network address such as golang.org:80 into a host
// string golang.org and a port integer 80.
func SplitHostPort(hostport string) (host, port string, err error) {
	// TODO: make this compliant with regular host/port splitting.
	// For now, use a simplified implementation.
	index := strings.LastIndexByte(hostport, ':')
	if index < 0 {
		return "", "", ErrInvalidHostPort
	}
	return hostport[:index], hostport[index+1:], nil
}
