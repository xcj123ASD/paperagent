package browse

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// blockedCIDRs 是除 Go 标准库预定义方法外，
// 还需要额外拦截的保留/特殊地址段。
//
// 标准库已覆盖（不需重复列）：
//   - IsLoopback: 127.0.0.0/8、::1
//   - IsPrivate: 10/8、172.16/12、192.168/16、fc00::/7
//   - IsLinkLocalUnicast: 169.254/16、fe80::/10
//   - IsLinkLocalMulticast: 224.0.0.0/24 等
//   - IsMulticast: 224/4、ff00::/8
//   - IsUnspecified: 0.0.0.0、::
var blockedCIDRs = mustParseCIDRs(
	"0.0.0.0/8",       // 本网络
	"100.64.0.0/10",   // CGNAT（运营商级 NAT）
	"192.0.0.0/24",    // IETF 协议分配
	"192.0.2.0/24",    // TEST-NET-1
	"198.18.0.0/15",   // 基准测试
	"198.51.100.0/24", // TEST-NET-2
	"203.0.113.0/24",  // TEST-NET-3
	"240.0.0.0/4",     // 保留段（含 255.255.255.255 广播）
)

func mustParseCIDRs(cidrs ...string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		p, err := netip.ParsePrefix(c)
		if err != nil {
			panic(fmt.Sprintf("browse: 非法 CIDR %q: %v", c, err))
		}
		out = append(out, p)
	}
	return out
}

// IsPrivateIP 判断 IP 是否属于私有 / 保留 / 回环 / 链路本地等
// 不应被外部请求访问的地址段。
//
// 覆盖范围：
//   - 回环（127.0.0.0/8、::1）
//   - 私有（10/8、172.16/12、192.168/16、fc00::/7）
//   - 链路本地（169.254/16 含云元数据 169.254.169.254、fe80::/10）
//   - 组播（224/4、ff00::/8）
//   - 未指定（0.0.0.0、::）
//   - 额外保留段（CGNAT、TEST-NET、基准测试、240/4 等）
//
// 传入 IPv4-mapped IPv6（::ffff:127.0.0.1）时先 Unmap 再判断，
// 防止通过 IPv6 表示绕过 IPv4 检查。
func IsPrivateIP(ip netip.Addr) bool {
	if !ip.IsValid() {
		return true // 无效 IP 一律拦截，安全默认
	}
	// 先转回 IPv4（若是 IPv4-mapped IPv6），避免 ::ffff:127.0.0.1 绕过
	ip = ip.Unmap()

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
		return true
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}

	for _, p := range blockedCIDRs {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// ErrNeedResolve 表示 host 是域名，需先 DNS 解析再调用 ValidateResolvedURL。
var ErrNeedResolve = errors.New("browse: 域名需解析后校验，请调用 ValidateResolvedURL")

// ValidateURL 校验 URL 是否允许访问。
//
// 校验顺序：
//  1. 协议必须是 https（唯一放行协议，http/ftp/file/javascript/data 全拦；http 可选放行）
//  2. URL 必须能被标准库解析
//  3. host 必须存在
//  4. host 若是 IP 字面量，直接过 IsPrivateIP
//  5. host 若是域名，返回 ErrNeedResolve（需先 DNS 解析再校验）
//
// 为何域名不能在 ValidateURL 阶段放行：
// 只校验域名格式无法防 DNS rebinding——域名可以解析到私有 IP。
// 必须先解析出 IP，再对每个 IP 过 IsPrivateIP。这是 ValidateResolvedURL 的工作。
func ValidateURL(rawURL string, allowInsecureHTTP bool) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("browse: URL 解析失败: %w", err)
	}

	// 1. 协议限制
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "https":
		// 放行
	case "http":
		if !allowInsecureHTTP {
			return fmt.Errorf("browse: 不允许 http 协议（仅 https），实际: %s", rawURL)
		}
	default:
		return fmt.Errorf("browse: 不允许的协议 %q（仅 https%s），实际: %s",
			scheme, map[bool]string{true: " / http", false: ""}[allowInsecureHTTP], rawURL)
	}

	// 2. host 必须存在
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("browse: URL 缺少 host: %s", rawURL)
	}

	// 3. host 是 IP 字面量 → 直接判断
	if ip, err := parseHostAsIP(host); err == nil {
		if IsPrivateIP(ip) {
			return fmt.Errorf("browse: 目标 IP 属于私有/保留段，拒绝访问: %s", host)
		}
		return nil
	}

	// 4. host 是域名 → 交给 ValidateResolvedURL 解析后再校验
	return ErrNeedResolve
}

// parseHostAsIP 尝试把 host 解析为 IP 字面量。
//
// 覆盖多种编码：
//   - 标准点分十进制：127.0.0.1
//   - 十进制整数：2130706433（=127.0.0.1）
//   - 八进制：0177.0.0.1
//   - 十六进制：0x7f.0.0.1
//   - IPv6：[::1] 已由 url.Parse 剥去方括号
//
// 标准库 netip.ParseAddr 只认标准写法，其他编码需先归一化。
func parseHostAsIP(host string) (netip.Addr, error) {
	// 先试标准写法
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip, nil
	}

	// 再试十进制整数（如 2130706433 → 127.0.0.1）
	if n, err := strconv.ParseUint(host, 10, 32); err == nil {
		return netip.AddrFrom4([4]byte{
			byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n),
		}), nil
	}

	// 八进制 / 十六进制 / 混合写法（如 0177.0.0.1、0x7f.0.0.1）
	// 拆成 4 段，每段按各自进制解析
	parts := strings.Split(host, ".")
	if len(parts) == 4 {
		var b [4]byte
		ok := true
		for i, p := range parts {
			v, err := parseNumericPart(p)
			if err != nil || v > 255 {
				ok = false
				break
			}
			b[i] = byte(v)
		}
		if ok {
			return netip.AddrFrom4(b), nil
		}
	}

	return netip.Addr{}, fmt.Errorf("browse: 非 IP 字面量: %q", host)
}

// parseNumericPart 解析一段数字，支持十进制 / 0 开头八进制 / 0x 开头十六进制。
func parseNumericPart(s string) (uint64, error) {
	if s == "" {
		return 0, fmt.Errorf("空段")
	}
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "0x"):
		return strconv.ParseUint(s[2:], 16, 32)
	case len(s) > 1 && s[0] == '0':
		return strconv.ParseUint(s[1:], 8, 32)
	default:
		return strconv.ParseUint(s, 10, 32)
	}
}

// blockedHostnames 是明确拒绝的 hostname，即使 DNS 解析到公网 IP 也拦。
// 这些名字有特殊含义（回环、元数据、内网服务），不应被外部请求访问。
var blockedHostnames = map[string]bool{
	"localhost":                true,
	"localhost.localdomain":    true,
	"ip6-localhost":            true,
	"ip6-loopback":             true,
	"metadata":                 true, // 部分云厂商的元数据 host
	"metadata.google.internal": true,
}

// ValidateResolvedURL 在 DNS 解析后对每个 IP 逐一校验，防 DNS rebinding。
//
// 为什么必须逐个 IP 校验而不是只查第一个：
// 域名可解析到多个 IP（轮询/负载均衡），若只查第一个，攻击者可让第一个是公网、
// 第二个是 127.0.0.1，绕过检查。必须对全部 IP 都过 IsPrivateIP。
//
// 解析失败（NXDOMAIN、超时）一律报错，不放行——安全默认。
//
// 调用顺序（重要）：
//  1. 先 ValidateURL：拦协议、拦 IP 字面量、域名返回 ErrNeedResolve
//  2. 若是 ErrNeedResolve，再调本函数：解析域名 + 逐 IP 校验
func ValidateResolvedURL(ctx context.Context, rawURL string, allowInsecureHTTP bool) error {
	// 先过基础校验（协议、host 存在、IP 字面量）
	err := ValidateURL(rawURL, allowInsecureHTTP)
	if err == nil {
		return nil // IP 字面量且已通过
	}
	if !errors.Is(err, ErrNeedResolve) {
		return err // 其他错误（协议、缺 host 等）直接返回
	}

	u, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return fmt.Errorf("browse: URL 解析失败: %w", parseErr)
	}
	host := u.Hostname()

	// hostname 黑名单（即使解析到公网 IP 也拦）
	if blockedHostnames[strings.ToLower(host)] {
		return fmt.Errorf("browse: hostname %q 在黑名单中，拒绝访问", host)
	}

	// DNS 解析
	ips, lookupErr := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if lookupErr != nil {
		return fmt.Errorf("browse: DNS 解析 %q 失败: %w", host, lookupErr)
	}
	if len(ips) == 0 {
		return fmt.Errorf("browse: DNS 解析 %q 无结果，拒绝访问", host)
	}

	// 逐个 IP 校验：任一私有即拦（防 DNS rebinding 多 IP 绕过）
	for _, ip := range ips {
		if IsPrivateIP(ip) {
			return fmt.Errorf("browse: 域名 %q 解析到私有 IP %s，拒绝访问（防 DNS rebinding）", host, ip)
		}
	}
	return nil
}
