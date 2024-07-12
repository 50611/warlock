package clientfactory

import (
	"context"
	"errors"
	"fmt"
	"github.com/50611/warlock/config"
	"google.golang.org/grpc"
	"log"
	"strings"
	"sync"
	"time"
)

var (
	errorTarget = errors.New("address is empty or invalid")
)

// PoolFactory object
type PoolFactory struct {
	config *config.Config
}
type condition = int

const (
	// Ready Can be used
	Ready condition = iota
	// Put Not available. Maybe later.
	Put
	// Destroy Failure occurs and cannot be restored
	Destroy
)

// NewPoolFactory get poolFactory
func NewPoolFactory(c *config.Config) *PoolFactory {
	return &PoolFactory{config: c}
}

// Passivate Action before releasing the resource
func (f *PoolFactory) Passivate(conn *grpc.ClientConn) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if conn.WaitForStateChange(ctx, 3) && conn.WaitForStateChange(ctx, 4) && conn.WaitForStateChange(ctx, 0) {
		return true, nil
	}
	return false, f.Destroy(conn)

}

// Activate Action taken after getting the resource
func (f *PoolFactory) Activate(conn *grpc.ClientConn) int {
	stat := conn.GetState()
	switch {
	case stat == 2:
		return Ready
	case stat == 0 || stat == 1 || stat == 3:
		return Put
	default:
		return Destroy
	}

}

// Destroy tears down the ClientConn and all underlying connections.
func (f *PoolFactory) Destroy(conn *grpc.ClientConn) error {
	return conn.Close()
}

// MakeConn Users are not recommended to use this API
func (f *PoolFactory) MakeConn(target string, ops ...grpc.DialOption) (*grpc.ClientConn, error) {
	if target == "" || strings.Index(target, ":") == -1 {
		return nil, errorTarget
	}
	if f.config.DynamicLink == true {
		return grpc.Dial(target, ops...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return grpc.DialContext(ctx, target, ops...)
}

// InitConn Initialize the create link
func (f *PoolFactory) InitConn(conns chan *grpc.ClientConn, ops ...grpc.DialOption) error {
	l := cap(conns) - len(conns)
	s := sync.WaitGroup{}
	s.Add(l)
	errlock := sync.Mutex{}
	errmsg := ""
	for i := 1; i <= l; i++ {
		go func() {
			defer s.Done()
			addr := f.config.GetTarget()
			cli, err := f.MakeConn(addr, ops...)
			if err != nil {
				errlock.Lock()
				errmsg = fmt.Sprintf("[grpc pool][%s] %s", addr, err.Error())
				log.Println(errmsg)
				errlock.Unlock()
			}
			conns <- cli
		}()

	}

	s.Wait()
	if len(errmsg) != 0 {
		return errors.New(errmsg)
	}
	return nil

}
