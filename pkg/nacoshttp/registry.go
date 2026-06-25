package nacoshttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/registry"
	"google.golang.org/grpc/resolver"
)

func init() {
	resolver.Register(&nacosResolverBuilder{})
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

// Registry 基于 Nacos HTTP API 的注册与发现实现，绕过 gRPC SDK
type Registry struct {
	baseURL     string
	namespaceID string
	groupName   string
	httpClient  *http.Client
}

// New 创建基于 HTTP API 的 Nacos Registry
func New(addr, namespaceID string) *Registry {
	return &Registry{
		baseURL:     "http://" + addr,
		namespaceID: namespaceID,
		groupName:   "DEFAULT_GROUP",
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (r *Registry) nsu(path string, params url.Values) string {
	params.Set("namespaceId", r.namespaceID)
	params.Set("groupName", r.groupName)
	return r.baseURL + path + "?" + params.Encode()
}

// Register 向 Nacos 注册服务（先清理旧实例，再注册持久实例）
func (r *Registry) Register(_ context.Context, si *registry.ServiceInstance) error {
	ip, port, err := parseEndpoint(si.Endpoints)
	if err != nil {
		return fmt.Errorf("nacoshttp register: %w", err)
	}

	// 先尝试清理可能存在的旧实例（ephemeral 或 persistent）
	r.deleteInstance(si.Name, ip, port)

	params := url.Values{}
	params.Set("serviceName", si.Name)
	params.Set("ip", ip)
	params.Set("port", fmt.Sprintf("%d", port))
	params.Set("healthy", "true")
	params.Set("ephemeral", "false")
	params.Set("weight", "1.0")

	resp, err := r.httpClient.Post(r.nsu("/nacos/v1/ns/instance", params), "application/x-www-form-urlencoded", nil)
	if err != nil {
		return fmt.Errorf("nacoshttp register: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("nacoshttp register: status=%d body=%s", resp.StatusCode, string(body))
	}
	return nil
}

func (r *Registry) deleteInstance(serviceName, ip string, port int) {
	params := url.Values{}
	params.Set("serviceName", serviceName)
	params.Set("ip", ip)
	params.Set("port", fmt.Sprintf("%d", port))
	params.Set("ephemeral", "true")
	req, _ := http.NewRequest(http.MethodDelete, r.nsu("/nacos/v1/ns/instance", params), nil)
	resp, err := r.httpClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

// Deregister 从 Nacos 注销服务
func (r *Registry) Deregister(_ context.Context, si *registry.ServiceInstance) error {
	ip, port, err := parseEndpoint(si.Endpoints)
	if err != nil {
		return fmt.Errorf("nacoshttp deregister: %w", err)
	}

	params := url.Values{}
	params.Set("serviceName", si.Name)
	params.Set("ip", ip)
	params.Set("port", fmt.Sprintf("%d", port))
	params.Set("ephemeral", "false")

	req, _ := http.NewRequest(http.MethodDelete, r.nsu("/nacos/v1/ns/instance", params), nil)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("nacoshttp deregister: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("nacoshttp deregister: status=%d body=%s", resp.StatusCode, string(body))
	}
	return nil
}

// GetService 从 Nacos 获取服务实例列表
func (r *Registry) GetService(_ context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	params := url.Values{}
	params.Set("serviceName", serviceName)
	params.Set("healthyOnly", "true")

	resp, err := r.httpClient.Get(r.nsu("/nacos/v1/ns/instance/list", params))
	if err != nil {
		return nil, fmt.Errorf("nacoshttp getservice: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("nacoshttp getservice: status=%d body=%s", resp.StatusCode, string(body))
	}

	var list nacosListResp
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("nacoshttp getservice decode: %w", err)
	}

	instances := make([]*registry.ServiceInstance, 0, len(list.Hosts))
	for _, h := range list.Hosts {
		if !h.Healthy {
			continue
		}
		instances = append(instances, &registry.ServiceInstance{
			ID:        fmt.Sprintf("%s-%s-%d", serviceName, h.IP, h.Port),
			Name:      serviceName,
			Endpoints: []string{fmt.Sprintf("grpc://%s:%d", h.IP, h.Port)},
		})
	}
	return instances, nil
}

// Watch 创建服务变更监听器（基于轮询）
func (r *Registry) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	w := &watcher{
		registry:    r,
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
	registry    *Registry
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
	instances, err := w.registry.GetService(context.Background(), w.serviceName)
	if err != nil {
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

// parseEndpoint 从 grpc://ip:port 格式提取 ip 和 port
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

	// replace wildcard/bind addresses with actual local IP
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = getLocalIP()
	}
	return host, port, nil
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}

// --- gRPC resolver 支持 ---

type nacosResolverBuilder struct{}

func (b *nacosResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	// discovery resolver handles this, no-op
	return nil, nil
}

func (b *nacosResolverBuilder) Scheme() string { return "discovery" }
