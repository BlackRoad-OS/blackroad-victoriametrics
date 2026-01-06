package main

import (
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/VictoriaMetrics/metrics"
)

var readDuration = metrics.NewSummaryExt(`vmauth_conn_read_op_duration_seconds`, time.Second*30, []float64{0.5, 0.9, 0.97, 0.99, 1})
var writeDuration = metrics.NewSummaryExt(`vmauth_conn_write_op_duration_seconds`, time.Second*30, []float64{0.5, 0.9, 0.97, 0.99, 1})

var readOpTimeout = flag.Duration("readOpTimeout", 0, "TODO")
var writeOpTimeout = flag.Duration("writeOpTimeout", 0, "TODO")

type connTimeoutTCPListener struct {
	ln net.Listener
}

func (ln *connTimeoutTCPListener) Accept() (net.Conn, error) {
	c, err := ln.ln.Accept()
	if err != nil {
		return nil, err
	}

	return &connTimeout{
		c: c,
	}, nil
}

func (ln *connTimeoutTCPListener) Close() error {
	return ln.ln.Close()
}

func (ln *connTimeoutTCPListener) Addr() net.Addr {
	return ln.ln.Addr()
}

type connTimeout struct {
	c net.Conn

	readDeadlineZero  bool
	writeDeadlineZero bool
}

func (conn *connTimeout) Read(b []byte) (n int, err error) {
	start := time.Now()
	defer readDuration.UpdateDuration(start)

	var deadlineSet bool
	if conn.readDeadlineZero && *readOpTimeout > 0 {
		deadlineSet = true
		if err := conn.c.SetReadDeadline(time.Now().Add(*readOpTimeout)); err != nil {
			return 0, fmt.Errorf("set read deadline to %v failed: %w", *readOpTimeout, err)
		}
	}

	n, err = conn.c.Read(b)
	if err != nil {
		return n, err
	}

	if deadlineSet {
		if err := conn.c.SetReadDeadline(time.Time{}); err != nil {
			return n, fmt.Errorf("unset read deadline failed: %w", err)
		}
	}

	return n, err
}

func (conn *connTimeout) Write(b []byte) (n int, err error) {
	start := time.Now()
	defer writeDuration.UpdateDuration(start)

	var deadlineSet bool
	if conn.writeDeadlineZero && *writeOpTimeout > 0 {
		deadlineSet = true
		if err := conn.c.SetWriteDeadline(time.Now().Add(*writeOpTimeout)); err != nil {
			return 0, fmt.Errorf("set write deadline to %v failed: %w", *writeOpTimeout, err)
		}
	}

	n, err = conn.c.Write(b)
	if err != nil {
		return n, err
	}

	if deadlineSet {
		if err := conn.c.SetWriteDeadline(time.Time{}); err != nil {
			return n, fmt.Errorf("unset write deadline failed: %w", err)
		}
	}

	return n, err
}

func (conn *connTimeout) Close() error {
	return conn.c.Close()
}

func (conn *connTimeout) LocalAddr() net.Addr {
	return conn.c.LocalAddr()
}

func (conn *connTimeout) RemoteAddr() net.Addr {
	return conn.c.RemoteAddr()
}

func (conn *connTimeout) SetDeadline(t time.Time) error {
	conn.readDeadlineZero = t.IsZero()
	conn.writeDeadlineZero = t.IsZero()

	return conn.c.SetDeadline(t)
}

func (conn *connTimeout) SetReadDeadline(t time.Time) error {
	conn.readDeadlineZero = t.IsZero()
	return conn.c.SetReadDeadline(t)
}

func (conn *connTimeout) SetWriteDeadline(t time.Time) error {
	conn.writeDeadlineZero = t.IsZero()
	return conn.c.SetWriteDeadline(t)
}
