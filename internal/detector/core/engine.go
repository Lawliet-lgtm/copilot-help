// Package core 检测引擎核心实现
package core

import (
	"fmt"
	"sync"
)

// DetectionEngine 检测引擎核心结构
type DetectionEngine struct {
	// 注册的检测器
	detectors map[string]Detector

	// 检测器配置
	configs map[string]interface{}

	// 互斥锁
	mu sync.RWMutex
}

// NewDetectionEngine 创建新的检测引擎
func NewDetectionEngine() *DetectionEngine {
	return &DetectionEngine{
		detectors: make(map[string]Detector),
		configs:   make(map[string]interface{}),
	}
}

// RegisterDetector 注册检测器
func (e *DetectionEngine) RegisterDetector(detector Detector) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	name := detector.GetName()
	if _, exists := e.detectors[name]; exists {
		return fmt.Errorf("detector with name %s already registered", name)
	}

	e.detectors[name] = detector
	return nil
}

// UnregisterDetector 注销检测器
func (e *DetectionEngine) UnregisterDetector(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.detectors, name)
	delete(e.configs, name)
}

// SetDetectorConfig 设置检测器配置
func (e *DetectionEngine) SetDetectorConfig(name string, config interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	detector, exists := e.detectors[name]
	if !exists {
		return fmt.Errorf("detector with name %s not found", name)
	}

	e.configs[name] = config
	return detector.Init(config)
}

// GetDetector 获取检测器
func (e *DetectionEngine) GetDetector(name string) (Detector, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	detector, exists := e.detectors[name]
	if !exists {
		return nil, fmt.Errorf("detector with name %s not found", name)
	}

	return detector, nil
}

// GetAllDetectors 获取所有检测器
func (e *DetectionEngine) GetAllDetectors() []Detector {
	e.mu.RLock()
	defer e.mu.RUnlock()

	detectors := make([]Detector, 0, len(e.detectors))
	for _, detector := range e.detectors {
		detectors = append(detectors, detector)
	}

	return detectors
}

// Detect 执行全量检测（串行）
func (e *DetectionEngine) Detect(path string) ([]*DetectionResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 按固定顺序执行检测器
	// 顺序：电子密级检测 → 密级标志检测 → 公文版式检测 → 文件哈希检测 → 关键词检测
	detectorOrder := []string{
		"electronic_secret_detect",
		"secret_level_detect",
		"official_format_detect",
		"file_hash_detect",
		"keyword_detect",
	}

	results := make([]*DetectionResult, 0)
	errors := make([]error, 0)

	// 串行执行检测
	for _, name := range detectorOrder {
		detector, exists := e.detectors[name]
		if !exists {
			continue // 跳过未注册的检测器
		}

		result, err := detector.Detect(path)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		if result != nil {
			results = append(results, result)

			// 短路逻辑：一旦检测到敏感内容，立即终止后续检测
			if result.Detected {
				break
			}
		}
	}

	// 如果有错误，但也有检测结果，仍然返回结果和错误
	if len(errors) > 0 {
		return results, fmt.Errorf("some detectors failed: %v", errors)
	}

	return results, nil
}

// Init 初始化检测引擎
func (e *DetectionEngine) Init() error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for name, detector := range e.detectors {
		config := e.configs[name]
		if err := detector.Init(config); err != nil {
			return fmt.Errorf("failed to initialize detector %s: %v", name, err)
		}
	}

	return nil
}
