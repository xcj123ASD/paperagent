package browse

import (
	"context"
	"errors"
	"net/netip"
	"testing"
)

// assertBlocked 断言给定 IP 字符串被拦截。
func assertBlocked(t *testing.T, s string) {
	t.Helper()
	ip, err := netip.ParseAddr(s)
	if err != nil {
		t.Fatalf("ParseAddr(%q) 失败: %v", s, err)
	}
	if !IsPrivateIP(ip) {
		t.Errorf("%s 应被拦截，实际放行", s)
	}
}

// assertAllowed 断言给定 IP 字符串被放行。
func assertAllowed(t *testing.T, s string) {
	t.Helper()
	ip, err := netip.ParseAddr(s)
	if err != nil {
		t.Fatalf("ParseAddr(%q) 失败: %v", s, err)
	}
	if IsPrivateIP(ip) {
		t.Errorf("%s 应放行，实际被拦截", s)
	}
}

// ---- IsPrivateIP ----

func TestIsPrivateIP_IPv4Private(t *testing.T) {
	for _, s := range []string{
		"10.0.0.1",
		"10.255.255.254",
		"172.16.0.1",
		"172.31.255.254",
		"192.168.0.1",
		"192.168.255.254",
	} {
		t.Run(s, func(t *testing.T) { assertBlocked(t, s) })
	}
}

func TestIsPrivateIP_Loopback(t *testing.T) {
	for _, s := range []string{
		"127.0.0.1",
		"127.0.0.2",
		"127.255.255.255",
	} {
		t.Run(s, func(t *testing.T) { assertBlocked(t, s) })
	}
}

func TestIsPrivateIP_LinkLocalCloudMetadata(t *testing.T) {
	for _, s := range []string{
		"169.254.169.254",
		"169.254.1.1",
	} {
		t.Run(s, func(t *testing.T) { assertBlocked(t, s) })
	}
}

func TestIsPrivateIP_SpecialRanges(t *testing.T) {
	for _, s := range []string{
		"0.0.0.0",
		"100.64.0.1",
		"192.0.0.1",
		"192.0.2.1",
		"198.18.0.1",
		"198.51.100.1",
		"203.0.113.1",
		"240.0.0.1",
		"255.255.255.255",
		"224.0.0.1",
	} {
		t.Run(s, func(t *testing.T) { assertBlocked(t, s) })
	}
}

func TestIsPrivateIP_IPv6(t *testing.T) {
	for _, s := range []string{
		"::1",
		"::",
		"fc00::1",
		"fd00::1",
		"fe80::1",
		"ff02::1",
	} {
		t.Run(s, func(t *testing.T) { assertBlocked(t, s) })
	}
}

func TestIsPrivateIP_IPv4MappedIPv6Bypass(t *testing.T) {
	for _, s := range []string{
		"::ffff:127.0.0.1",
		"::ffff:10.0.0.1",
		"::ffff:192.168.1.1",
		"::ffff:169.254.169.254",
	} {
		t.Run(s, func(t *testing.T) { assertBlocked(t, s) })
	}
}

func TestIsPrivateIP_PublicAllowed(t *testing.T) {
	for _, s := range []string{
		"8.8.8.8",
		"1.1.1.1",
		"93.184.216.34",
		"2606:4700:4700::1111",
	} {
		t.Run(s, func(t *testing.T) { assertAllowed(t, s) })
	}
}

func TestIsPrivateIP_Invalid(t *testing.T) {
	var ip netip.Addr
	if !IsPrivateIP(ip) {
		t.Error("无效 IP 应拦截（安全默认），实际放行")
	}
}

func TestIsPrivateIP_Boundary(t *testing.T) {
	assertBlocked(t, "10.0.0.0")
	assertBlocked(t, "10.255.255.255")
	assertBlocked(t, "172.16.0.0")
	assertBlocked(t, "172.31.255.255")
	assertAllowed(t, "172.15.0.1")
	assertAllowed(t, "172.32.0.1")
}

// ---- ValidateURL ----

func TestValidateURL_HTTPSPublicIP(t *testing.T) {
	if err := ValidateURL("https://8.8.8.8/path", false); err != nil {
		t.Errorf("公网 IP + https 应放行，实际: %v", err)
	}
}

func TestValidateURL_HTTPSPrivateIP(t *testing.T) {
	for _, u := range []string{
		"https://127.0.0.1/",
		"https://10.0.0.1/",
		"https://192.168.1.1/",
		"https://169.254.169.254/latest/meta-data/",
		"https://[::1]/",
		"https://[::ffff:127.0.0.1]/",
	} {
		t.Run(u, func(t *testing.T) {
			if err := ValidateURL(u, false); err == nil {
				t.Errorf("%s 应被拦截，实际放行", u)
			}
		})
	}
}

func TestValidateURL_DecimalIPBypass(t *testing.T) {
	for _, u := range []string{
		"https://2130706433/",
		"https://0x7f.0.0.1/",
		"https://0177.0.0.1/",
	} {
		t.Run(u, func(t *testing.T) {
			if err := ValidateURL(u, false); err == nil {
				t.Errorf("%s 应被识别为 127.0.0.1 并拦截，实际放行", u)
			}
		})
	}
}

func TestValidateURL_ProtocolRejected(t *testing.T) {
	for _, u := range []string{
		"file:///etc/passwd",
		"ftp://example.com/",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"http://example.com/",
	} {
		t.Run(u, func(t *testing.T) {
			if err := ValidateURL(u, false); err == nil {
				t.Errorf("%s 协议应被拦截，实际放行", u)
			}
		})
	}
}

func TestValidateURL_HTTPAllowedWhenFlagged(t *testing.T) {
	if err := ValidateURL("http://8.8.8.8/", true); err != nil {
		t.Errorf("allowInsecureHTTP=true 时 http 应放行，实际: %v", err)
	}
	if err := ValidateURL("http://127.0.0.1/", true); err == nil {
		t.Error("allowInsecureHTTP=true 时私有 IP 仍应拦截")
	}
}

func TestValidateURL_DomainNeedsResolve(t *testing.T) {
	err := ValidateURL("https://example.com/", false)
	if !errors.Is(err, ErrNeedResolve) {
		t.Errorf("域名应返回 ErrNeedResolve，实际: %v", err)
	}
}

func TestValidateURL_MissingHost(t *testing.T) {
	if err := ValidateURL("https:///path", false); err == nil {
		t.Error("缺 host 应报错")
	}
}

func TestValidateURL_InvalidURL(t *testing.T) {
	if err := ValidateURL("://bad", false); err == nil {
		t.Error("非法 URL 应报错")
	}
}

// ---- ValidateResolvedURL ----

// 注意：TestValidateResolvedURL_PublicDomain 会真做 DNS 查询（只查不连），
// 依赖网络。CI 环境若 DNS 不稳可能偶发失败——是网络问题不是代码 bug。
// -short 模式跳过。

func TestValidateResolvedURL_IPLiteralPassesThrough(t *testing.T) {
	if err := ValidateResolvedURL(context.Background(), "https://8.8.8.8/", false); err != nil {
		t.Errorf("公网 IP 应放行，实际: %v", err)
	}
	if err := ValidateResolvedURL(context.Background(), "https://127.0.0.1/", false); err == nil {
		t.Error("私有 IP 应拦截")
	}
}

func TestValidateResolvedURL_ProtocolError(t *testing.T) {
	if err := ValidateResolvedURL(context.Background(), "file:///etc/passwd", false); err == nil {
		t.Error("file:// 应拦截")
	}
	if err := ValidateResolvedURL(context.Background(), "http://example.com/", false); err == nil {
		t.Error("http 默认应拦截")
	}
}

func TestValidateResolvedURL_BlockedHostname(t *testing.T) {
	for _, u := range []string{
		"https://localhost/",
		"https://LOCALHOST/",
		"https://metadata/",
		"https://metadata.google.internal/",
	} {
		t.Run(u, func(t *testing.T) {
			if err := ValidateResolvedURL(context.Background(), u, false); err == nil {
				t.Errorf("%s 应在 hostname 黑名单中被拦截", u)
			}
		})
	}
}

func TestValidateResolvedURL_ResolutionFails(t *testing.T) {
	err := ValidateResolvedURL(context.Background(), "https://this-domain-definitely-not-exists-xyz123.example/", false)
	if err == nil {
		t.Error("DNS 解析失败应报错")
	}
}

func TestValidateResolvedURL_PublicDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("short 模式跳过 DNS 查询")
	}
	if err := ValidateResolvedURL(context.Background(), "https://example.com/", false); err != nil {
		t.Errorf("example.com 应放行，实际: %v", err)
	}
}
