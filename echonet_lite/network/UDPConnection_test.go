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
	port, err := getFreePort()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	receiver, err := CreateUDPConnection(ctx, net.IPv4zero, port, net.IPv4bcast, UDPConnectionOptions{DefaultTimeout: 1 * time.Second})
	require.NoError(t, err)
	defer receiver.Close()

	payload := []byte("broadcast test")
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		sender, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			errCh <- err
			return
		}
		defer sender.Close()
		rc, err := sender.SyscallConn()
		if err != nil {
			errCh <- err
			return
		}
		rc.Control(func(fd uintptr) {
			_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
		})
		_, err = sender.WriteToUDP(payload, &net.UDPAddr{IP: net.IPv4bcast, Port: port})
		errCh <- err
	}()

	recvCtx, recvCancel := context.WithTimeout(ctx, 2*time.Second)
	defer recvCancel()

	assert.NoError(t, <-errCh)
	data, src, err := receiver.Receive(recvCtx)
	require.NoError(t, err)
	assert.Equal(t, payload, data)
	assert.NotNil(t, src)
}

// TestUDPConnection_ReceiveMulticast verifies that UDPConnection can receive multicast packets.
func TestUDPConnection_ReceiveMulticast(t *testing.T) {
	const multicastIPStr = "224.0.23.0"
	multicastIP := net.ParseIP(multicastIPStr).To4()
	require.NotNil(t, multicastIP, "invalid multicast IP")

	port, err := getFreePort()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	receiver, err := CreateUDPConnection(ctx, net.IPv4zero, port, multicastIP, UDPConnectionOptions{DefaultTimeout: 1 * time.Second})
	require.NoError(t, err)
	defer receiver.Close()

	payload := []byte("multicast test")
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		sender, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			errCh <- err
			return
		}
		defer sender.Close()
		rc, err := sender.SyscallConn()
		if err != nil {
			errCh <- err
			return
		}
		rc.Control(func(fd uintptr) {
			_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_MULTICAST_LOOP, 1)
			_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_MULTICAST_TTL, 1)
		})
		_, err = sender.WriteToUDP(payload, &net.UDPAddr{IP: multicastIP, Port: port})
		errCh <- err
	}()

	recvCtx, recvCancel := context.WithTimeout(ctx, 2*time.Second)
	defer recvCancel()

	assert.NoError(t, <-errCh)
	data, src, err := receiver.Receive(recvCtx)
	require.NoError(t, err)
	assert.Equal(t, payload, data)
	assert.NotNil(t, src)
}
