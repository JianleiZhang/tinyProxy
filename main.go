package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"time"

	"github.com/things-go/go-socks5"
)

var (
	ErrAddressNotSupported = errors.New("address type not supported")
)

// 自定义 DNS 解析器（实现 NameResolver 接口）
type CustomResolver struct {
	upstreamDNS string // 自定义DNS服务器，格式如 "8.8.8.8:53"
}

func (r *CustomResolver) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 创建自定义解析器
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 3 * time.Second}
			return d.DialContext(ctx, "udp", r.upstreamDNS) // 指向自定义DNS服务器
		},
	}

	// 执行DNS查询
	addrs, err := resolver.LookupIPAddr(ctx, name)
	if err != nil {
		return ctx, nil, err
	}

	// 优先返回IPv4地址
	for _, addr := range addrs {
		if ipv4 := addr.IP.To4(); ipv4 != nil {
			return ctx, ipv4, nil
		}
	}

	// 没有IPv4则返回第一个IPv6地址
	if len(addrs) > 0 {
		return ctx, addrs[0].IP, nil
	}

	return ctx, nil, ErrAddressNotSupported
}

func main() {
	// 创建SOCKS5服务器
	server := socks5.NewServer(
		socks5.WithResolver(&CustomResolver{
			upstreamDNS: "223.5.5.5:53", // 使用Google DNS
		}),
		socks5.WithLogger(socks5.NewLogger(log.New(os.Stdout, "socks5: ", log.LstdFlags))),
	)

	// 启动服务
	if err := server.ListenAndServe("tcp", ":1080"); err != nil {
		panic(err)
	}
}
