package app

import (
	"fmt"
	"io"
	"mime/multipart"
	"sync"
	"sync/atomic"
)

type Headers struct {
	*multipart.FileHeader
	ContentType string
}

type Handle struct {
	mssg    chan Mssg
	once    sync.Once
	wg      sync.WaitGroup
	entered atomic.Bool
}

type Conn struct {
	pr         *io.PipeReader
	pw         *io.PipeWriter
	headers    chan Headers
	progress   chan int64
	joined     chan struct{}
	joinedOnce sync.Once
	sender     *Handle
	receiver   *Handle
}

func NewConn() *Conn {
	pr, pw := io.Pipe()
	return &Conn{
		pr:       pr,
		pw:       pw,
		headers:  make(chan Headers),
		progress: make(chan int64),
		joined:   make(chan struct{}),
		sender:   &Handle{mssg: make(chan Mssg)},
		receiver: &Handle{mssg: make(chan Mssg)},
	}
}

func (c *Conn) Enter(peer Peer) error {
	switch peer {
	case PeerSender:
		if c.sender.entered.Load() {
			return fmt.Errorf("sender already joined connection")
		}
		c.joinedOnce.Do(func() { close(c.joined) })
		return nil
	case PeerReceiver:
		if c.receiver.entered.Load() {
			return fmt.Errorf("receiver already joined connection")
		}
		c.joinedOnce.Do(func() { close(c.joined) })
		return nil
	}
	return fmt.Errorf("failed to join connection as unknown")
}

func (c *Conn) CanEnter(peer Peer) bool {
	switch peer {
	case PeerSender:
		return !c.sender.entered.Load()
	case PeerReceiver:
		return !c.receiver.entered.Load()
	}
	return false
}

func (c *Conn) AnyJoined() <-chan struct{} { return c.joined }

func (c *Conn) Mssg(peer Peer) <-chan Mssg {
	switch peer {
	case PeerSender:
		return c.sender.mssg
	case PeerReceiver:
		return c.receiver.mssg
	default:
		return nil
	}
}

func (c *Conn) Broadcast(m Mssg) {
	c.sender.wg.Add(1)
	go func() {
		defer c.sender.wg.Done()
		c.sender.mssg <- m
	}()
	c.receiver.wg.Add(1)
	go func() {
		defer c.receiver.wg.Done()
		c.receiver.mssg <- m
	}()
}

func (c *Conn) CloseWriter() {
	c.pw.Close()
	c.sender.once.Do(func() {
		c.sender.wg.Wait()
		close(c.sender.mssg)
	})
}

func (c *Conn) CloseReader() {
	c.pr.Close()
	c.receiver.once.Do(func() {
		c.receiver.wg.Wait()
		close(c.receiver.mssg)
	})
}

func (c *Conn) Close() {
	c.CloseWriter()
	c.CloseReader()
}

func (c *Conn) SendHeaders(h Headers) { c.headers <- h }

func (c *Conn) Send(r io.Reader) (written int64, err error) {
	n, err := io.Copy(c.pw, &ProgressReader{Reader: r, Progress: c.progress})
	c.pw.CloseWithError(err)
	return n, err
}

func (c *Conn) ReceiveHeaders() Headers { return <-c.headers }

func (c *Conn) Receive(w io.Writer) (written int64, err error) {
	return io.Copy(w, c.pr)
}

func (c *Conn) Progress() <-chan int64 { return c.progress }

type ProgressReader struct {
	io.Reader
	Progress chan<- int64
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	if err != nil {
		return n, err
	}
	go func() { pr.Progress <- int64(n) }()
	return n, err
}
