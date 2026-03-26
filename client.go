package goesl

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client - In case you need to do inbound dialing against freeswitch server in order to originate call or see
// sofia statuses or whatever else you came up with
type Client struct {
	SocketConnection

	Proto   string `json:"freeswitch_protocol"`
	Addr    string `json:"freeswitch_addr"`
	Passwd  string `json:"freeswitch_password"`
	Timeout int    `json:"freeswitch_connection_timeout"`

	// 内部状态管理
	mu            sync.RWMutex
	connected     bool
	reconnecting  bool
	stopHeartbeat chan struct{}
}

// 添加状态检查方法
func (c *Client) isConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *Client) setConnected(state bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = state
}

func (c *Client) isReconnecting() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.reconnecting
}

func (c *Client) setReconnecting(state bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reconnecting = state
}

// 心跳检测
func (c *Client) startHeartbeat() {
	c.stopHeartbeat = make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := c.Send("api status"); err != nil {
					Debug("Heartbeat failed: %v", err)
					c.setConnected(false)
					go c.reconnect()
				}
			case <-c.stopHeartbeat:
				return
			}
		}
	}()
}

// 重连逻辑
func (c *Client) reconnect() {
	if c.isReconnecting() {
		return
	}
	c.setReconnecting(true)
	defer c.setReconnecting(false)

	for !c.isConnected() {
		Debug("Attempting immediate reconnection...")

		// 关闭旧连接
		c.Close()

		// 立即尝试重新建立连接
		if err := c.EstablishConnection(); err != nil {
			Debug("Connection failed: %v", err)
			continue  // 立即重试，不等待
		}

		// 重新认证
		if err := c.Authenticate(); err != nil {
			Debug("Authentication failed: %v", err)
			continue  // 立即重试，不等待
		}

		c.setConnected(true)
		Debug("Successfully reconnected")

		// 重新启动心跳检测
		c.startHeartbeat()
		return
	}
}

// EstablishConnection - Will attempt to establish connection against freeswitch and create new SocketConnection
func (c *Client) EstablishConnection() error {
	conn, err := c.Dial(c.Proto, c.Addr, time.Duration(c.Timeout)*time.Second)
	if err != nil {
		return err
	}

	c.SocketConnection = SocketConnection{
		Conn: conn,
		err:  make(chan error),
		m:    make(chan *Message),
	}

	return nil
}

// Authenticate - Method used to authenticate client against freeswitch. In case of any errors durring so
// we will return error.
func (c *Client) Authenticate() error {

	m, err := newMessage(bufio.NewReaderSize(c, ReadBufferSize), false)
	if err != nil {
		Error(ECouldNotCreateMessage, err)
		return err
	}

	cmr, err := m.tr.ReadMIMEHeader()
	if err != nil && err.Error() != "EOF" {
		Error(ECouldNotReadMIMEHeaders, err)
		return err
	}

	Debug("A: %v\n", cmr)

	if cmr.Get("Content-Type") != "auth/request" {
		Error(EUnexpectedAuthHeader, cmr.Get("Content-Type"))
		return fmt.Errorf(EUnexpectedAuthHeader, cmr.Get("Content-Type"))
	}

	s := "auth " + c.Passwd + "\r\n\r\n"
	_, err = io.WriteString(c, s)
	if err != nil {
		return err
	}

	am, err := m.tr.ReadMIMEHeader()
	if err != nil && err.Error() != "EOF" {
		Error(ECouldNotReadMIMEHeaders, err)
		return err
	}

	if am.Get("Reply-Text") != "+OK accepted" {
		Error(EInvalidPassword, c.Passwd)
		return fmt.Errorf(EInvalidPassword, c.Passwd)
	}

	return nil
}

// NewClient - Will initiate new client that will establish connection and attempt to authenticate
// against connected freeswitch server
func NewClient(host string, port uint, passwd string, timeout int) (*Client, error) {
	client := Client{
		Proto:        "tcp",
		Addr:         net.JoinHostPort(host, strconv.Itoa(int(port))),
		Passwd:       passwd,
		Timeout:      timeout,
		connected:    false,
		reconnecting: false,
	}

	err := client.EstablishConnection()
	if err != nil {
		return nil, err
	}

	err = client.Authenticate()
	if err != nil {
		client.Close()
		return nil, err
	}

	client.connected = true
	client.startHeartbeat()
	return &client, nil
}

// Close - Will close the connection to freeswitch server
func (c *Client) Close() error {
	// 只在确实需要关闭时才完全关闭
	if c.isConnected() {
		if c.stopHeartbeat != nil {
			close(c.stopHeartbeat)
			c.stopHeartbeat = nil
		}
		c.setConnected(false)
		return c.SocketConnection.Close()
	}
	return nil
}

// Read - Will read data from freeswitch server
func (c *Client) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if err != nil {
		if isConnectionError(err) {
			c.setConnected(false)
			go c.reconnect()
		}
	}
	return n, err
}

// Write - Will write data to freeswitch server
func (c *Client) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if err != nil {
		if isConnectionError(err) {
			c.setConnected(false)
			go c.reconnect()
		}
	}
	return n, err
}

// 连接错误检测
func isConnectionError(err error) bool {
	if err == io.EOF {
		return true
	}
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() ||
			strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "broken pipe") ||
			strings.Contains(err.Error(), "reset by peer") ||
			strings.Contains(err.Error(), "connection closed") ||
			strings.Contains(err.Error(), "use of closed network connection")
	}
	return false
}
