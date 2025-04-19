package network

import (
	"context"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getFreePort returns an available UDP port by letting the OS assign one.
func getFreePort() (int, error) {
	addr, err := net.ResolveUDPAddr("udp", "localhost:0")
	if err != nil {
		return 0, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).Port, nil
}

// TestUDPConnection_ReceiveBroadcast verifies that UDPConnection can receive broadcast packets.
func TestUDPConnection_ReceiveBroadcast(t *testing.T) {
	// Obtain a free port
	port, err := getFreePort()
	require.NoError(t, err)

	// Create receiver UDPConnection on 0.0.0.0:<port>
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := UDPConnectionOptions{DefaultTimeout: 1 * time.Second}
	receiver, err := CreateUDPConnection(ctx, net.IPv4zero, port, net.IPv4bcast, opts)
	require.NoError(t, err)
	defer receiver.Close()

	// Prepare payload
	payload := []byte("broadcast test")

	// Send broadcast in a goroutine
	sendErrCh := make(chan error, 1)
	go func() {
		defer close(sendErrCh)
		senderConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			sendErrCh <- err
			return
		}
		defer senderConn.Close()

		// Enable broadcast on the socket via syscall
		if rc, err := senderConn.SyscallConn(); err != nil {
			sendErrCh <- err
			return
		} else {
			rc.Control(func(fd uintptr) {
				_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
			})
		}

		// Broadcast address
		baddr := &net.UDPAddr{IP: net.IPv4bcast, Port: port}
		_, err = senderConn.WriteToUDP(payload, baddr)
		sendErrCh <- err
	}()

	// Receive
	recvCtx, recvCancel := context.WithTimeout(ctx, 2*time.Second)
	defer recvCancel()

	data, addr, err := receiver.Receive(recvCtx)
	assert.NoError(t, <-sendErrCh)
	require.NoError(t, err)
	assert.Equal(t, payload, data)
	assert.NotNil(t, addr)

	// Verify sender IP is one of the host's non-loopback interfaces
	addrs, err := net.InterfaceAddrs()
	require.NoError(t, err)
	found := false
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.Equal(addr.IP) {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "sender IP %s should be a local interface", addr.IP.String())
}
