package vm

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// The following constants define TTY flags in the status register.
const (
	TTYIn = 1 << iota
	TTYOut
)

// The following errors may be emitted by the TTY implementation.
var (
	ErrTTYDetach = errors.New("tty: detach")
)

// SerialTTY is a serial TTY.
//
// The user of this struct is supposed to create a new instance by
// calling TTYAcceptConn. The user shall defer calling Close. The user
// shall otherwise not manipulate the SerialTTY and store it inside
// the TTY field of the VM. The VM shall manage the TTY.
type SerialTTY struct {
	conn  net.Conn // control conn
	inr   uint32   // input register
	outr  uint32   // output register
	statr uint32   // status register
}

// TTYAcceptConn waits for a controlling TCP connection to attach
// to the console. Once there is a control connection, this function
// returns with the serial TTY console instance.
func TTYAcceptConn() (*SerialTTY, error) {
	nl, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	log.Printf("tty: waiting for console to attach on %s/tcp...", nl.Addr())
	conn, err := nl.Accept()
	if err != nil {
		return nil, err
	}
	return &SerialTTY{conn: conn}, nil
}

// Close closes the underlying connection.
func (tty *SerialTTY) Close() error {
	return tty.conn.Close()
}

// LocalAddr returns the address where we're listening.
func (tty *SerialTTY) LocalAddr() net.Addr {
	return tty.conn.LocalAddr()
}

// InRegister implements TTY.InRegister.
func (tty *SerialTTY) InRegister() (*uint32, error) {
	return &tty.inr, nil
}

// OutRegister implements TTY.OutOutRegister.
func (tty *SerialTTY) OutRegister() (*uint32, error) {
	return &tty.outr, nil
}

// StatusRegister implements TTY.StatusRegister.
func (tty *SerialTTY) StatusRegister() (*uint32, error) {
	return &tty.statr, nil
}

// InterruptPending implements TTY.InterruptPending. This function may
// block for a bunch of milliseconds if there is no input from the conn
// but will not wait forever and will not block the VM forever.
func (tty *SerialTTY) InterruptPending() (bool, error) {
	// The timeout is such that we certainly can read/write if we have data
	// however, if we don't have data, we don't block the VM.
	tty.conn.SetDeadline(time.Now().Add(time.Millisecond))
	if (tty.statr & TTYOut) != 0 {
		var c [1]byte
		c[0] = byte(tty.outr & 0xff)
		if _, err := tty.conn.Write(c[:]); err != nil {
			// We're basically polling the connection every time and we don't
			// declare an interrupt when we can't do I/O.
			if strings.HasSuffix(err.Error(), "i/o timeout") {
				return false, nil
			}
			return false, fmt.Errorf("%w: %s", ErrTTYDetach, err.Error())
		}
		tty.statr &^= TTYOut // byte has been sent
	}
	if (tty.statr & TTYIn) == 0 {
		var c [1]byte
		if _, err := tty.conn.Read(c[:]); err != nil {
			// We're basically polling the connection every time and we don't
			// declare an interrupt when we can't do I/O.
			if strings.HasSuffix(err.Error(), "i/o timeout") {
				return false, nil
			}
			return false, fmt.Errorf("%w: %s", ErrTTYDetach, err.Error())
		}
		tty.statr |= TTYIn // byte has been received
		tty.inr = uint32(c[0])
	}
	return (tty.statr & (TTYIn | TTYOut)) != 0, nil
}

var _ TTY = &SerialTTY{}
