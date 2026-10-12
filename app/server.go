package app

import "time"

type Server struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	MaxBodyBytes      int64
}

var defaultServer = Server{
	ReadHeaderTimeout: 10 * time.Second,
	ReadTimeout:       30 * time.Second,
	WriteTimeout:      60 * time.Second,
	IdleTimeout:       120 * time.Second,
	MaxHeaderBytes:    1 << 20,
	MaxBodyBytes:      10 << 20,
}

func (s Server) withDefaults() Server {
	if s.ReadHeaderTimeout <= 0 {
		s.ReadHeaderTimeout = defaultServer.ReadHeaderTimeout
	}
	if s.ReadTimeout <= 0 {
		s.ReadTimeout = defaultServer.ReadTimeout
	}
	if s.WriteTimeout <= 0 {
		s.WriteTimeout = defaultServer.WriteTimeout
	}
	if s.IdleTimeout <= 0 {
		s.IdleTimeout = defaultServer.IdleTimeout
	}
	if s.MaxHeaderBytes <= 0 {
		s.MaxHeaderBytes = defaultServer.MaxHeaderBytes
	}
	if s.MaxBodyBytes <= 0 {
		s.MaxBodyBytes = defaultServer.MaxBodyBytes
	}
	return s
}
