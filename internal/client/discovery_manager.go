package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"crosswire/internal/models"
	"crosswire/internal/transport"
)

// DiscoveryManager 服务发现管理器
type DiscoveryManager struct {
	client *Client
	ctx    context.Context
	cancel context.CancelFunc

	// 已发现的服务器
	servers      map[string]*DiscoveredServer
	serversMutex sync.RWMutex

	// 统计信息
	stats      DiscoveryStats
	statsMutex sync.RWMutex
}

// DiscoveredServer 已发现的服务器信息
type DiscoveredServer struct {
	ID              string
	Name            string
	Address         string
	Port            int
	TransportMode   models.TransportMode
	ProtocolVersion string
	MemberCount     int
	ServerPublicKey []byte
	DiscoveredAt    time.Time
	LastSeenAt      time.Time
	TXT             map[string]string // TXT记录
}

// DiscoveryStats 服务发现统计
type DiscoveryStats struct {
	TotalScans        int64
	ServersDiscovered int64
	LastScanTime      time.Time
	mutex             sync.RWMutex
}

// NewDiscoveryManager 创建服务发现管理器
func NewDiscoveryManager(client *Client) *DiscoveryManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &DiscoveryManager{
		client:  client,
		ctx:     ctx,
		cancel:  cancel,
		servers: make(map[string]*DiscoveredServer),
	}
}

// Start 启动服务发现管理器
func (dm *DiscoveryManager) Start() error {
	dm.client.logger.Info("[DiscoveryManager] Starting...")
	dm.client.logger.Info("[DiscoveryManager] Started successfully")
	return nil
}

// Stop 停止服务发现管理器
func (dm *DiscoveryManager) Stop() error {
	dm.client.logger.Info("[DiscoveryManager] Stopping...")
	dm.cancel()
	dm.client.logger.Info("[DiscoveryManager] Stopped")
	return nil
}

// Discover 扫描局域网中的服务器
func (dm *DiscoveryManager) Discover(timeout time.Duration) ([]*DiscoveredServer, error) {
	dm.client.logger.Info("[DiscoveryManager] Starting discovery scan...")

	// 更新统计
	dm.statsMutex.Lock()
	dm.stats.TotalScans++
	dm.stats.LastScanTime = time.Now()
	dm.statsMutex.Unlock()

	// 使用Transport的Discover方法
	if dm.client.transport == nil {
		return nil, fmt.Errorf("transport not initialized")
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(dm.ctx, timeout)
	defer cancel()

	// 扫描
	discoveries, err := dm.discoverWithTimeout(ctx)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}

	// 处理发现的服务器
	servers := make([]*DiscoveredServer, 0, len(discoveries))
	now := time.Now()

	dm.serversMutex.Lock()
	for _, peer := range discoveries {
		server := dm.peerInfoToServer(peer)
		server.LastSeenAt = now

		// 如果是新服务器，设置DiscoveredAt
		if existing, ok := dm.servers[server.ID]; ok {
			server.DiscoveredAt = existing.DiscoveredAt
		} else {
			server.DiscoveredAt = now
			dm.statsMutex.Lock()
			dm.stats.ServersDiscovered++
			dm.statsMutex.Unlock()
		}

		dm.servers[server.ID] = server
		servers = append(servers, server)
	}
	dm.serversMutex.Unlock()

	dm.client.logger.Info("[DiscoveryManager] Discovery completed: %d servers found", len(servers))

	return servers, nil
}

// discoverWithTimeout 带超时的服务发现
func (dm *DiscoveryManager) discoverWithTimeout(ctx context.Context) ([]*transport.PeerInfo, error) {
	// 创建结果channel
	resultCh := make(chan []*transport.PeerInfo, 1)
	errCh := make(chan error, 1)

	// 获取超时时间
	deadline, _ := ctx.Deadline()
	timeout := time.Until(deadline)

	// 异步执行发现
	go func() {
		peers, err := dm.client.transport.Discover(timeout)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- peers
	}()

	// 等待结果或超时
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("discovery timeout")
	case err := <-errCh:
		return nil, err
	case peers := <-resultCh:
		return peers, nil
	}
}

// peerInfoToServer 将PeerInfo转换为DiscoveredServer
func (dm *DiscoveryManager) peerInfoToServer(peer *transport.PeerInfo) *DiscoveredServer {
	server := &DiscoveredServer{
		ID:              peer.ID,
		Name:            peer.ID, // 使用ID作为名称
		Address:         peer.Address,
		TransportMode:   models.TransportAuto, // 默认值
		ProtocolVersion: "1.0",
		TXT:             make(map[string]string),
	}

	// 解析元数据（如果存在）
	// TODO: 当transport.PeerInfo添加Metadata字段后，解析它

	return server
}

// GetDiscoveredServers 获取已发现的服务器列表
func (dm *DiscoveryManager) GetDiscoveredServers() []*DiscoveredServer {
	dm.serversMutex.RLock()
	defer dm.serversMutex.RUnlock()

	servers := make([]*DiscoveredServer, 0, len(dm.servers))
	for _, server := range dm.servers {
		servers = append(servers, server)
	}

	return servers
}

// GetServerByID 根据ID获取服务器
func (dm *DiscoveryManager) GetServerByID(serverID string) (*DiscoveredServer, bool) {
	dm.serversMutex.RLock()
	defer dm.serversMutex.RUnlock()

	server, ok := dm.servers[serverID]
	return server, ok
}

// ClearServers 清除已发现的服务器列表
func (dm *DiscoveryManager) ClearServers() {
	dm.serversMutex.Lock()
	defer dm.serversMutex.Unlock()

	dm.servers = make(map[string]*DiscoveredServer)
	dm.client.logger.Debug("[DiscoveryManager] Cleared server list")
}

// CleanupStaleServers 清理过期的服务器（超过一定时间未见）
func (dm *DiscoveryManager) CleanupStaleServers(maxAge time.Duration) {
	dm.serversMutex.Lock()
	defer dm.serversMutex.Unlock()

	now := time.Now()
	removed := 0

	for id, server := range dm.servers {
		if now.Sub(server.LastSeenAt) > maxAge {
			delete(dm.servers, id)
			removed++
		}
	}

	if removed > 0 {
		dm.client.logger.Debug("[DiscoveryManager] Cleaned up %d stale servers", removed)
	}
}

// GetStats 获取统计信息
func (dm *DiscoveryManager) GetStats() DiscoveryStats {
	dm.statsMutex.RLock()
	defer dm.statsMutex.RUnlock()

	// 复制统计信息（避免复制锁）
	return DiscoveryStats{
		TotalScans:        dm.stats.TotalScans,
		ServersDiscovered: dm.stats.ServersDiscovered,
		LastScanTime:      dm.stats.LastScanTime,
	}
}

// DiscoverAuto 自动发现服务器（根据配置的传输模式）
func (dm *DiscoveryManager) DiscoverAuto() ([]*DiscoveredServer, error) {
	// 默认5秒超时
	timeout := 5 * time.Second

	// 根据传输模式调整超时
	if dm.client.config.TransportMode == models.TransportMDNS {
		timeout = 10 * time.Second // mDNS需要更长时间
	} else if dm.client.config.TransportMode == models.TransportARP {
		timeout = 3 * time.Second // ARP比较快
	}

	return dm.Discover(timeout)
}

// StartPeriodicDiscovery 启动定期扫描
func (dm *DiscoveryManager) StartPeriodicDiscovery(interval time.Duration) {
	dm.client.logger.Info("[DiscoveryManager] Starting periodic discovery (interval: %v)", interval)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-dm.ctx.Done():
				dm.client.logger.Debug("[DiscoveryManager] Periodic discovery stopped")
				return
			case <-ticker.C:
				if _, err := dm.DiscoverAuto(); err != nil {
					dm.client.logger.Warn("[DiscoveryManager] Periodic discovery failed: %v", err)
				}
			}
		}
	}()
}

// StopPeriodicDiscovery 停止定期扫描
func (dm *DiscoveryManager) StopPeriodicDiscovery() {
	dm.cancel()
	dm.client.logger.Debug("[DiscoveryManager] Periodic discovery cancelled")
}
