package collection

import (
	"context"
	"go.uber.org/zap"
	"net"

	"github.com/siriusfreak/free-proxy-gun/log"
)

type ProxyType int

const (
	ProxyTypeHTTP ProxyType = iota
	ProxyTypeHTTPS
	ProxyTypeSOCKS4
	ProxyTypeSOCKS5
)

func ParseProxyType(str string) ProxyType {
	switch str {
	case "http":
		return ProxyTypeHTTP
	case "https":
		return ProxyTypeHTTPS
	case "socks4":
		return ProxyTypeSOCKS4
	case "socks5":
		return ProxyTypeSOCKS5
	}
	return ProxyTypeHTTP
}

type AnonymousLevel int

const (
	AnonymousLevelTransparent AnonymousLevel = iota
	AnonymousLevelAnonymous
	AnonymousLevelElite
)

func ParseAnonymousLevel(str string) AnonymousLevel {
	switch str {
	case "transparent":
		return AnonymousLevelTransparent
	case "anonymous":
		return AnonymousLevelAnonymous
	case "elite":
		return AnonymousLevelElite
	}
	return AnonymousLevelTransparent
}

type Proxy struct {
	IP             net.IP
	Port           int
	Type           ProxyType
	AnonymousLevel AnonymousLevel
	Country        string
}

type Collector interface {
	GetName() string
	Collect(ctx context.Context) ([]Proxy, error)
}

func Collect(ctx context.Context, Collectors []Collector) ([]Proxy, error) {
	res := make([]Proxy, 0)
	for _, collector := range Collectors {
		proxies, err := collector.Collect(ctx)
		if err != nil {
			log.Warn(ctx, "error while collecting proxies", zap.Error(err), zap.String("collector", collector.GetName()))
			continue
		}
		res = append(res, proxies...)
	}
	return res, nil
}
