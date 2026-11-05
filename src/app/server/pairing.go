package main

// 配对（Pairing）：把「其他设备接入这台 NAS 上的 OpenCode」做成界面里的一个入口。
//
// 三个接口挂在门户路径下（同时注册带前缀 / 不带前缀两种）：
//   GET  /__pair/api/state   直连端口、地址、令牌、服务状态
//   POST /__pair/api/pair    调上游 /api/pair 生成一次性配对码，返回链接与二维码
//   POST /__pair/api/enable  开/关局域网直连端口
//
// 直连端口本身由 startTCP 提供：浏览器用令牌换 cookie，客户端可用
// Basic（服务端密码）或带配对码的 /auth/connect/<code> 交换会话。

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

// registerPairRoutes 注册配对相关路由（前缀 / 非前缀各一份）
func registerPairRoutes(m *http.ServeMux) {
	for _, p := range []string{"", PREFIX} {
		m.HandleFunc(p+"/__pair/api/state", pairState)
		m.HandleFunc(p+"/__pair/api/pair", pairCreate)
		m.HandleFunc(p+"/__pair/api/enable", pairEnable)
	}
}

func pairJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// lanURLs 返回其他设备可访问的局域网地址（按常见私网段优先排序）
func lanURLs() []string {
	port := tcpPort()
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{"http://127.0.0.1:" + port}
	}
	for _, it := range ifaces {
		if it.Flags&net.FlagUp == 0 || it.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := it.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				ip4 := ipnet.IP.To4()
				if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
					continue
				}
				ips = append(ips, ip4.String())
			}
		}
	}
	sort.SliceStable(ips, func(i, j int) bool {
		return ipRank(ips[i]) < ipRank(ips[j])
	})
	if len(ips) == 0 {
		ips = append(ips, "127.0.0.1")
	}
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, "http://"+net.JoinHostPort(ip, port))
	}
	return out
}

func ipRank(ip string) int {
	switch {
	case strings.HasPrefix(ip, "192.168."):
		return 0
	case strings.HasPrefix(ip, "10."):
		return 1
	case strings.HasPrefix(ip, "172."):
		return 2
	default:
		return 3
	}
}

func demoPortFile() string { return filepath.Join(OC_HOME, "port_tcp") }

func tcpEnabled() bool {
	_, err := os.Stat(demoPortFile())
	return err == nil
}

// pairState 返回面板需要的全部状态
func pairState(w http.ResponseWriter, r *http.Request) {
	pairJSON(w, 200, map[string]any{
		"online":   probe(),
		"enabled":  tcpEnabled(),
		"port":     tcpPort(),
		"token":    ensureToken(),
		"urls":     lanURLs(),
		"version":  VERSION,
		"hostname": hostName(),
	})
}

func hostName() string {
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "fnOS"
}

// pairCreate 生成一次性配对码（调用上游 /api/pair，失败时给出明确原因）
func pairCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		pairJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	pw, _ := servePass.Load().(string)
	if pw == "" {
		pairJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "服务尚未就绪（还没拿到访问密码），请稍后重试"})
		return
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/api/pair", OC_PORT), strings.NewReader("{}"))
	if err != nil {
		pairJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	req.SetBasicAuth("opencode", pw)
	req.Header.Set("Content-Type", "application/json")

	cli := &http.Client{Timeout: 12 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		pairJSON(w, http.StatusBadGateway, map[string]any{"error": "无法连接 OpenCode 服务：" + err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if resp.StatusCode != 200 {
		pairJSON(w, resp.StatusCode, map[string]any{"error": strings.TrimSpace(string(body))})
		return
	}

	var pr struct {
		Code      string `json:"code"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &pr); err != nil || pr.Code == "" {
		pairJSON(w, 502, map[string]any{"error": "配对接口返回异常：" + strings.TrimSpace(string(body))})
		return
	}

	urls := lanURLs()
	link := urls[0] + "/auth/connect/" + pr.Code
	// 二维码载荷与官方 App 一致：{"code":…,"urls":[…]}
	payload, _ := json.Marshal(map[string]any{"code": pr.Code, "urls": urls})
	png, qerr := qrcode.Encode(string(payload), qrcode.Medium, 512)
	qr := ""
	if qerr == nil {
		qr = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
	}

	pairJSON(w, 200, map[string]any{
		"code":       pr.Code,
		"expires_in": pr.ExpiresIn,
		"urls":       urls,
		"link":       link,
		"qr":         qr,
		"enabled":    tcpEnabled(),
	})
}

// pairEnable 开/关局域网直连端口。开启后立即生效，不需要重启应用。
func pairEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		pairJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var in struct {
		On   bool   `json:"on"`
		Port string `json:"port"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&in)

	if !in.On {
		_ = os.Remove(demoPortFile())
		pairJSON(w, 200, map[string]any{"ok": true, "enabled": false, "port": tcpPort()})
		return
	}

	port := strings.TrimSpace(in.Port)
	if port == "" {
		port = "4096"
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1024 || n > 65535 {
		pairJSON(w, 400, map[string]any{"error": "端口必须在 1024-65535 之间"})
		return
	}
	if err := os.WriteFile(demoPortFile(), []byte(port+"\n"), 0600); err != nil {
		pairJSON(w, 500, map[string]any{"error": "写入端口配置失败：" + err.Error()})
		return
	}
	ensureTCPStarted()
	pairJSON(w, 200, map[string]any{
		"ok":      true,
		"enabled": true,
		"port":    port,
		"urls":    lanURLs(),
		"token":   ensureToken(),
	})
}
