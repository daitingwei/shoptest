package nacos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/gateway/discovery"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
)

func init() {
	discovery.Register("nacos", New)
}

// Nacos 返回的实例结构
type nacosHost struct {
	IP      string  `json:"ip"`
	Port    int     `json:"port"`
	Healthy bool    `json:"healthy"`
	Weight  float64 `json:"weight"`
}

type nacosListResp struct {
	Hosts []nacosHost `json:"hosts"`
}

// Discovery 基于 Nacos HTTP API 的服务发现实现
type Discovery struct {
	baseURL     string
	namespaceID string
	groupName   string
	httpClient  *http.Client
}

// New 根据 DSN 创建 Nacos Discovery 实例
// DSN 格式: nacos://127.0.0.1:8848?namespaceId=public&groupName=DEFAULT_GROUP
func New(dsn *url.URL) (registry.Discovery, error) {
	if dsn.Scheme != "nacos" {
		return nil, fmt.Errorf("invalid scheme %q, expected nacos", dsn.Scheme)
	}

	namespaceID := dsn.Query().Get("namespaceId")
	if namespaceID == "" {
		namespaceID = "public"
	}
	groupName := dsn.Query().Get("groupName")
	if groupName == "" {
		groupName = "DEFAULT_GROUP"
	}

	baseURL := "http://" + dsn.Host
	if dsn.User != nil {
		password, _ := dsn.User.Password()
		baseURL = "http://" + dsn.User.Username() + ":" + password + "@" + dsn.Host
	}

	return &Discovery{
		baseURL:     baseURL,
		namespaceID: namespaceID,
		groupName:   groupName,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}, nil
}

func (d *Discovery) nsu(path string, params url.Values) string {
	params.Set("namespaceId", d.namespaceID)
	params.Set("groupName", d.groupName)
	return d.baseURL + path + "?" + params.Encode()
}

// GetService 从 Nacos 获取服务实例列表
func (d *Discovery) GetService(_ context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	params := url.Values{}
	params.Set("serviceName", serviceName)
	params.Set("healthyOnly", "true")

	resp, err := d.httpClient.Get(d.nsu("/nacos/v1/ns/instance/list", params))
	if err != nil {
		return nil, fmt.Errorf("nacos getservice: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("nacos getservice: status=%d body=%s", resp.StatusCode, string(body))
	}

	var list nacosListResp
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("nacos getservice decode: %w", err)
	}

	instances := make([]*registry.ServiceInstance, 0, len(list.Hosts))
	for _, h := range list.Hosts {
		if !h.Healthy {
			continue
		}
		instances = append(instances, &registry.ServiceInstance{
			ID:        fmt.Sprintf("%s-%s-%d", serviceName, h.IP, h.Port),
			Name:      serviceName,
			Endpoints: []string{fmt.Sprintf("http://%s:%d", h.IP, h.Port)},
			Metadata: map[string]string{
				"weight": fmt.Sprintf("%f", h.Weight),
			},
		})
	}
	return instances, nil
}

// Watch 创建服务变更监听器（基于轮询）
func (d *Discovery) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	w := &watcher{
		discovery:   d,
		serviceName: serviceName,
		interval:    10 * time.Second,
		ch:          make(chan []*registry.ServiceInstance, 1),
		done:        make(chan struct{}),
	}
	go w.poll(ctx)
	return w, nil
}

// watcher 基于轮询的 Nacos 服务监听器
type watcher struct {
	discovery   *Discovery
	serviceName string
	interval    time.Duration
	ch          chan []*registry.ServiceInstance
	done        chan struct{}
	once        sync.Once
	lastResult  []*registry.ServiceInstance
	mu          sync.Mutex
}

func (w *watcher) poll(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// 首次立即获取
	w.fetch()

	for {
		select {
		case <-ticker.C:
			w.fetch()
		case <-ctx.Done():
			return
		case <-w.done:
			return
		}
	}
}

func (w *watcher) fetch() {
	instances, err := w.discovery.GetService(context.Background(), w.serviceName)
	if err != nil {
		log.Errorf("nacos watcher fetch error: %v", err)
		return
	}
	w.mu.Lock()
	w.lastResult = instances
	w.mu.Unlock()
	select {
	case w.ch <- instances:
	default:
	}
}

// Next 阻塞等待服务实例变更
func (w *watcher) Next() ([]*registry.ServiceInstance, error) {
	select {
	case instances := <-w.ch:
		return instances, nil
	case <-w.done:
		w.mu.Lock()
		defer w.mu.Unlock()
		if w.lastResult != nil {
			return w.lastResult, nil
		}
		return nil, fmt.Errorf("watcher stopped")
	}
}

// Stop 停止监听
func (w *watcher) Stop() error {
	w.once.Do(func() {
		close(w.done)
	})
	return nil
}

// parseEndpoint 从 http://ip:port 格式提取 ip 和 port
func parseEndpoint(endpoints []string) (string, int, error) {
	if len(endpoints) == 0 {
		return "", 0, fmt.Errorf("no endpoints")
	}
	raw := endpoints[0]
	if !strings.Contains(raw, "://") {
		return "", 0, fmt.Errorf("invalid endpoint format: %s", raw)
	}
	addr := strings.SplitN(raw, "://", 2)[1]

	// strip brackets from IPv6 addresses like [::]
	addr = strings.TrimPrefix(addr, "[")
	addr = strings.Replace(addr, "]", "", 1)

	colon := strings.LastIndex(addr, ":")
	if colon < 0 {
		return "", 0, fmt.Errorf("no port in endpoint: %s", raw)
	}
	var port int
	if _, err := fmt.Sscanf(addr[colon+1:], "%d", &port); err != nil {
		return "", 0, fmt.Errorf("invalid port: %s", addr[colon+1:])
	}
	host := addr[:colon]
	return host, port, nil
}
