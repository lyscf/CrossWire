package transport

import (
	"crosswire/internal/models"
	"fmt"
)

// Factory 传输层工厂
type Factory struct{}

// NewFactory 创建工厂实例
func NewFactory() *Factory {
	return &Factory{}
}

// Create 创建传输层实例
func (f *Factory) Create(mode TransportMode) (Transport, error) {
	switch mode {
	case TransportModeHTTPS:
		return NewHTTPSTransport(), nil

	case TransportModeARP:
		return NewARPTransport(), nil

	case TransportModeMDNS:
		return NewMDNSTransport(), nil

	default:
		return nil, fmt.Errorf("unknown transport mode: %s", mode)
	}
}

// CreateWithConfig 创建并初始化传输层
func (f *Factory) CreateWithConfig(mode TransportMode, config *Config) (Transport, error) {
	transport, err := f.Create(mode)
	if err != nil {
		return nil, err
	}

	if err := transport.Init(config); err != nil {
		return nil, fmt.Errorf("failed to initialize transport: %w", err)
	}

	return transport, nil
}

// GetSupportedModes 获取支持的传输模式列表
func (f *Factory) GetSupportedModes() []models.TransportMode {
	return []models.TransportMode{
		models.TransportHTTPS,
		models.TransportARP,
		models.TransportMDNS,
	}
}

// IsModeSupported 检查传输模式是否支持
func (f *Factory) IsModeSupported(mode models.TransportMode) bool {
	for _, m := range f.GetSupportedModes() {
		if m == mode {
			return true
		}
	}
	return false
}
