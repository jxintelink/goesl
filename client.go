// Copyright 2015 Nevio Vesic
// Please check out LICENSE file for more information about what you CAN and what you CANNOT do!
// Basically in short this is a free software for you to do whatever you want to do BUT copyright must be included!
// I didn't write all of this code so you could say it's yours.
// MIT License

package goesl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
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

	// DisableAutoReconnect 为 true 时使用 SocketConnection.Handle（断线即关闭，不自动重拨）。
	DisableAutoReconnect bool `json:"-"`

	// HandleContext 控制自动重连生命周期，nil 时使用 context.Background()。
	HandleContext context.Context `json:"-"`

	// AutoReconnectInterval 为两次重连尝试之间的等待时间；<=0 时默认 3s。
	AutoReconnectInterval time.Duration `json:"-"`

	// ResubscribeOnReconnect 非空时，每次重连成功后会通过 Send 发送该命令（常见如 "events json ALL"）。
	ResubscribeOnReconnect string `json:"-"`
}

// EstablishConnection - Will attempt to establish connection against freeswitch and create new SocketConnection
func (c *Client) EstablishConnection() error {
	conn, err := c.Dial(c.Proto, c.Addr, time.Duration(c.Timeout*int(time.Second)))
	if err != nil {
		return err
	}

	c.SocketConnection = SocketConnection{
		Conn: conn,
		err:  make(chan error, 1),
		m:    make(chan *Message),
	}

	return nil
}

// Authenticate - Method used to authenticate client against freeswitch. In case of any errors durring so
// we will return error.
func (c *Client) Authenticate() error {
	return c.authenticate()
}

func (c *Client) authenticate() error {

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

// AutoReconnectConfig 控制 HandleWithAutoReconnect 在断线后的重连行为。
type AutoReconnectConfig struct {
	// MaxAttempts 单次断线后，连续重拨+认证的最大尝试次数；<=0 表示在 ctx 取消前无限重试。
	MaxAttempts int
	// Interval 两次重连尝试之间的等待时间，建议 >0。
	Interval time.Duration
	// OnReconnected 每次重连成功、即将恢复读流前调用，可在此重新执行 event 订阅等。
	OnReconnected func()
}

// reconnect 重新拨号并完成认证。replaceChannels 为 true 时重建 err/m（与 Reconnect 一致）；
// 为 false 时保留原有通道，供断线自动重连时 ReadMessage 仍从同一 c.m 取消息。
func (c *Client) reconnect(replaceChannels bool) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.Conn != nil {
		_ = c.Conn.Close()
		c.Conn = nil
	}

	conn, err := c.Dial(c.Proto, c.Addr, time.Duration(c.Timeout*int(time.Second)))
	if err != nil {
		return err
	}

	c.Conn = conn
	if replaceChannels {
		c.err = make(chan error, 1)
		c.m = make(chan *Message)
	}

	if err := c.authenticate(); err != nil {
		_ = c.Conn.Close()
		c.Conn = nil
		return err
	}

	return nil
}

// Reconnect 关闭当前连接（若存在），重新拨号并完成 ESL 认证，并重建内部消息通道。
// 典型用法：ReadMessage/Handle 返回错误或检测到断线后，在同一线程或已停止并发读写的时机调用。
// 不要在仍有 goroutine 对同一 Client 执行 ReadMessage 时并发调用 Reconnect，否则可能永远阻塞在旧通道上。
func (c *Client) Reconnect() error {
	return c.reconnect(true)
}

// ReconnectWithRetry 在 Reconnect 失败时按 interval 等待后重试。
// maxAttempts > 0 时最多尝试 maxAttempts 次（每次均执行一次完整的关闭/拨号/认证）；
// maxAttempts <= 0 时在 ctx 取消前无限重试。interval 建议大于 0，以避免 tight loop。
func (c *Client) ReconnectWithRetry(ctx context.Context, maxAttempts int, interval time.Duration) error {
	var lastErr error
	for attempt := 1; ; attempt++ {
		lastErr = c.reconnect(true)
		if lastErr == nil {
			return nil
		}
		if maxAttempts > 0 && attempt >= maxAttempts {
			return fmt.Errorf("goesl: reconnect failed after %d attempts: %w", maxAttempts, lastErr)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("goesl: reconnect cancelled (after %d attempts): %w; last error: %v", attempt, ctx.Err(), lastErr)
		case <-time.After(interval):
		}
	}
}

func (c *Client) reconnectWithRetryKeepChannels(ctx context.Context, maxAttempts int, interval time.Duration) error {
	var lastErr error
	for attempt := 1; ; attempt++ {
		lastErr = c.reconnect(false)
		if lastErr == nil {
			return nil
		}
		if maxAttempts > 0 && attempt >= maxAttempts {
			return fmt.Errorf("goesl: reconnect failed after %d attempts: %w", maxAttempts, lastErr)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("goesl: reconnect cancelled (after %d attempts): %w; last error: %v", attempt, ctx.Err(), lastErr)
		case <-time.After(interval):
		}
	}
}

// HandleWithAutoReconnect 在后台持续读取 ESL 并将消息写入 c.m（与 Handle 相同消费方式）。
// 读失败时自动关闭旧连接并按 cfg 重拨、认证，不替换 err/m，ReadMessage 可在重连后继续阻塞等待新消息。
// OnReconnected 中请重新发送 events 等订阅，否则可能收不到事件。
// ctx 取消时关闭连接并结束；重连耗尽后向 c.err 写入错误并退出。
func (c *Client) HandleWithAutoReconnect(ctx context.Context, cfg AutoReconnectConfig) {
	go func() {
		defer func() {
			_ = c.Close()
		}()
		for {
			if ctx.Err() != nil {
				return
			}
			rbuf := bufio.NewReaderSize(c, ReadBufferSize)
			readDead := make(chan error, 1)
			go func() {
				for {
					msg, err := newMessage(rbuf, true)
					if err != nil {
						readDead <- err
						return
					}
					select {
					case c.m <- msg:
					case <-ctx.Done():
						readDead <- ctx.Err()
						return
					}
				}
			}()
			var readErr error
			select {
			case <-ctx.Done():
				_ = c.Close()
				return
			case readErr = <-readDead:
			}
			if errors.Is(readErr, context.Canceled) || errors.Is(readErr, context.DeadlineExceeded) {
				return
			}
			Warning("goesl: ESL read failed (%v), reconnecting ...", readErr)
			if err := c.reconnectWithRetryKeepChannels(ctx, cfg.MaxAttempts, cfg.Interval); err != nil {
				select {
				case c.err <- err:
				default:
				}
				return
			}
			if cfg.OnReconnected != nil {
				cfg.OnReconnected()
			}
		}
	}()
}

// Handle 对 Inbound Client 默认启用断线自动重连（见 HandleWithAutoReconnect），本方法立即返回。
// 可在首次连上后照常 Send("events ...")；若设置了 ResubscribeOnReconnect，重连成功后会自动再次 Send。
// 需要原版单次连接读循环时，设 DisableAutoReconnect=true（内部仍会起 goroutine，行为同旧版 go + SocketConnection.Handle）。
func (c *Client) Handle() {
	if c.DisableAutoReconnect {
		go func() {
			c.SocketConnection.Handle()
		}()
		return
	}

	ctx := c.HandleContext
	if ctx == nil {
		ctx = context.Background()
	}

	interval := c.AutoReconnectInterval
	if interval <= 0 {
		interval = 3 * time.Second
	}

	resub := c.ResubscribeOnReconnect
	cfg := AutoReconnectConfig{
		MaxAttempts: 0,
		Interval:    interval,
		OnReconnected: func() {
			if resub == "" {
				return
			}
			if err := c.Send(resub); err != nil {
				Error("goesl: ResubscribeOnReconnect Send: %v", err)
			}
		},
	}
	c.HandleWithAutoReconnect(ctx, cfg)
}

// NewClient - Will initiate new client that will establish connection and attempt to authenticate
// against connected freeswitch server
func NewClient(host string, port uint, passwd string, timeout int) (*Client, error) {
	client := Client{
		Proto:   "tcp", // Let me know if you ever need this open up lol
		Addr:    net.JoinHostPort(host, strconv.Itoa(int(port))),
		Passwd:  passwd,
		Timeout: timeout,
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

	return &client, nil
}
