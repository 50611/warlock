package config

import (
	"math/rand"
	"sync"
	"time"
)

// Config Configuring the properties of the pool
type Config struct {
	Lock *sync.RWMutex
	//地址
	ServerAdds *[]string
	//最大数量
	MaxCap int64
	//是否直接链接
	DynamicLink bool
	//是否超过
	OverflowCap bool
	//获取超时时间
	AcquireTimeout time.Duration
	//connectTimeout
	ConnectTimeout time.Duration

	GetTargetFunc GetTargetFunc
	Lazy          bool
}

type GetTargetFunc func(c *Config) string

func init() {

	//rand.Seed(time.Now().Unix())
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

// GetTarget If there are multiple services  will be randomly selected
func (c *Config) GetTarget() string {
	c.Lock.RLock()
	defer c.Lock.RUnlock()
	if c.GetTargetFunc != nil {
		return c.GetTargetFunc(c)
	}
	cLen := len(*c.ServerAdds)
	if cLen <= 0 {
		return ""
	}
	return (*c.ServerAdds)[rand.Int()%cLen]
}
