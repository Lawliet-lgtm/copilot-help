// Package file_hash 文件哈希检测子模块
package file_hash

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"linuxFileWatcher/internal/config"
	"linuxFileWatcher/internal/detector/core"
	"linuxFileWatcher/internal/detector/policy"
	"linuxFileWatcher/internal/logger"
	"linuxFileWatcher/internal/model"
	"linuxFileWatcher/internal/security/integrity"
	"linuxFileWatcher/internal/storage"
)

// Detector 文件哈希检测器
type Detector struct {
	// 检测器名称
	name string

	// 检测器版本
	version string

	// 使用 map 实现 O(1) 复杂度的快速查找
	// 按 RuleType 分类的哈希映射
	// 0: MD5, 1: SM3
	hashToRuleID map[int]map[string]int64

	// 规则ID到规则详情的映射
	ruleMap map[int64]model.HashDetectRule

	// 策略管理器
	policyManager *policy.Manager
}

// NewDetector 创建新的文件哈希检测器
func NewDetector() *Detector {
	// 初始化策略管理器
	var policiesPath string
	var policyManager *policy.Manager

	// 使用defer和recover捕获配置未初始化的panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				// 配置未初始化，使用默认值
				logger.Warn("Config not initialized, using default policies path",
					"error", r,
				)
				policiesPath = "./policies"
				policyManager = policy.NewManager(policiesPath)

				logger.Info("Created file hash detector with default path",
					"name", "file_hash_detector",
					"version", "1.0.0",
					"policies_path", policiesPath,
				)
			}
		}()

		// 尝试获取配置
		policiesPath = config.Get().Scanner.PoliciesPath
		policyManager = policy.NewManager(policiesPath)

		logger.Info("Created file hash detector",
			"name", "file_hash_detector",
			"version", "1.0.0",
			"policies_path", policiesPath,
		)
	}()

	// 确保policyManager已初始化
	if policyManager == nil {
		// 如果recover没有触发（可能是其他错误），使用默认值
		policiesPath = "./policies"
		policyManager = policy.NewManager(policiesPath)

		logger.Info("Created file hash detector with default path (fallback)",
			"name", "file_hash_detector",
			"version", "1.0.0",
			"policies_path", policiesPath,
		)
	}

	return &Detector{
		name:    "file_hash_detector",
		version: "1.0.0",
		hashToRuleID: map[int]map[string]int64{
			0: make(map[string]int64), // MD5
			1: make(map[string]int64), // SM3
		},
		ruleMap:       make(map[int64]model.HashDetectRule),
		policyManager: policyManager,
	}
}

// GetName 返回检测器名称
func (d *Detector) GetName() string {
	return d.name
}

// GetVersion 返回检测器版本
func (d *Detector) GetVersion() string {
	return d.version
}

// Init 初始化检测器
func (d *Detector) Init(config interface{}) error {
	// 初始化映射
	d.hashToRuleID = map[int]map[string]int64{
		0: make(map[string]int64), // MD5
		1: make(map[string]int64), // SM3
	}
	d.ruleMap = make(map[int64]model.HashDetectRule)

	// 如果传入了配置参数，使用传入的配置
	if config != nil {
		if hashConfig, ok := config.(*model.HashDetectConfig); ok {
			// 编译哈希值到规则ID的映射，按RuleType分类
			for _, rule := range hashConfig.Rules {
				// 确保RuleType对应的映射存在
				if _, exists := d.hashToRuleID[rule.RuleType]; !exists {
					d.hashToRuleID[rule.RuleType] = make(map[string]int64)
				}
				d.hashToRuleID[rule.RuleType][rule.RuleContent] = rule.RuleID
				d.ruleMap[rule.RuleID] = rule
			}
			return nil
		}
	}

	// 否则从本地文件加载策略
	return d.loadPolicy()
}

// loadPolicy 从本地文件加载策略
func (d *Detector) loadPolicy() error {
	var config model.HashDetectConfig
	if err := d.policyManager.LoadPolicy(model.ModuleMD5Detect, &config); err != nil {
		return err
	}

	// 编译哈希值到规则ID的映射，按RuleType分类
	for _, rule := range config.Rules {
		// 确保RuleType对应的映射存在
		if _, exists := d.hashToRuleID[rule.RuleType]; !exists {
			d.hashToRuleID[rule.RuleType] = make(map[string]int64)
		}
		d.hashToRuleID[rule.RuleType][rule.RuleContent] = rule.RuleID
		d.ruleMap[rule.RuleID] = rule
	}

	return nil
}

// Detect 执行检测操作
func (d *Detector) Detect(path string) (*core.DetectionResult, error) {
	// 检查文件是否存在
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 检查文件大小
	if fileInfo.Size() == 0 {
		return &core.DetectionResult{
			DetectorName: d.name,
			Detected:     false,
			Matches:      []core.MatchDetail{},
		}, nil
	}

	// 性能优化：限制文件大小，避免对超大文件进行哈希计算
	// 这里设置为 100MB，可以根据实际情况调整
	const maxFileSize = 100 * 1024 * 1024 // 100MB
	if fileInfo.Size() > maxFileSize {
		logger.Info("File too large, skipping hash detection",
			"path", path,
			"size", fileInfo.Size(),
			"max_size", maxFileSize,
		)
		return &core.DetectionResult{
			DetectorName: d.name,
			Detected:     false,
			Matches:      []core.MatchDetail{},
		}, nil
	}

	// 获取文件信息
	fileName := fileInfo.Name()
	fileSize := int(fileInfo.Size())

	// 计算文件的哈希值
	var md5Hash, sm3Hash string
	var md5Err, sm3Err error

	// 根据策略中的 RuleType 确定需要检测的哈希类型
	needMD5 := len(d.hashToRuleID[0]) > 0
	needSM3 := len(d.hashToRuleID[1]) > 0

	if needMD5 {
		md5Hash, md5Err = computeFileMD5(path)
		if md5Err != nil {
			logger.Error("Failed to compute MD5 hash",
				"path", path,
				"error", md5Err,
			)
		}
	}

	if needSM3 {
		sm3Hash, sm3Err = integrity.ComputeFileSM3(path)
		if sm3Err != nil {
			logger.Error("Failed to compute SM3 hash",
				"path", path,
				"error", sm3Err,
			)
		}
	}

	// 构建匹配详情和告警记录
	matches := []core.MatchDetail{}
	alerts := []*model.AlertRecord{}

	// 检查 MD5 哈希
	if needMD5 && md5Hash != "" {
		if ruleID, ok := d.hashToRuleID[0][md5Hash]; ok {
			// 获取规则详情
			rule, ruleExists := d.ruleMap[ruleID]
			ruleDesc := "MD5 Hash Match"
			if ruleExists && rule.RuleDesc != "" {
				ruleDesc = rule.RuleDesc
			}

			// 构建匹配详情
			matches = append(matches, core.MatchDetail{
				MatchType:   "file_hash",
				Content:     md5Hash,
				Location:    "file",
				RuleID:      ruleID,
				RuleDesc:    ruleDesc,
				AlertType:   int(model.AlertTypeOther),
				FileSummary: "敏感文件哈希匹配",
				FileDesc:    fmt.Sprintf("文件 %s 的 MD5 哈希值匹配敏感文件规则", fileName),
				FileLevel:   4,
			})

			// 构建告警记录
			alert := model.NewAlertRecord(fmt.Sprintf("alert_%d", time.Now().UnixNano()))
			alert.Time = time.Now().Format("2006-01-02 15:04:05")
			alert.RuleID = ruleID
			alert.RuleDesc = ruleDesc
			alert.FilterType = 0
			alert.FileSummary = "敏感文件哈希匹配"
			alert.AlertType = model.AlertTypeOther
			alert.FileMD5 = md5Hash
			alert.FilePath = path
			alert.FileName = fileName
			alert.FileSize = fileSize
			alert.HighlightText = md5Hash
			alert.FileDesc = fmt.Sprintf("文件 %s 的 MD5 哈希值匹配敏感文件规则", fileName)
			alert.FileLevel = 4

			alerts = append(alerts, alert)
		}
	}

	// 检查 SM3 哈希
	if needSM3 && sm3Hash != "" {
		if ruleID, ok := d.hashToRuleID[1][sm3Hash]; ok {
			// 获取规则详情
			rule, ruleExists := d.ruleMap[ruleID]
			ruleDesc := "SM3 Hash Match"
			if ruleExists && rule.RuleDesc != "" {
				ruleDesc = rule.RuleDesc
			}

			// 构建匹配详情
			matches = append(matches, core.MatchDetail{
				MatchType:   "file_hash",
				Content:     sm3Hash,
				Location:    "file",
				RuleID:      ruleID,
				RuleDesc:    ruleDesc,
				AlertType:   int(model.AlertTypeOther),
				FileSummary: "敏感文件哈希匹配",
				FileDesc:    fmt.Sprintf("文件 %s 的 SM3 哈希值匹配敏感文件规则", fileName),
				FileLevel:   4,
			})

			// 构建告警记录
			alert := model.NewAlertRecord(fmt.Sprintf("alert_%d", time.Now().UnixNano()))
			alert.Time = time.Now().Format("2006-01-02 15:04:05")
			alert.RuleID = ruleID
			alert.RuleDesc = ruleDesc
			alert.FilterType = 0
			alert.FileSummary = "敏感文件哈希匹配"
			alert.AlertType = model.AlertTypeOther
			alert.FileMD5 = md5Hash // 如果MD5计算失败，这里可能为空
			alert.FilePath = path
			alert.FileName = fileName
			alert.FileSize = fileSize
			alert.HighlightText = sm3Hash
			alert.FileDesc = fmt.Sprintf("文件 %s 的 SM3 哈希值匹配敏感文件规则", fileName)
			alert.FileLevel = 4

			alerts = append(alerts, alert)
		}
	}

	// 存储告警记录
	stores := storage.GetStores()
	if stores != nil {
		for _, alert := range alerts {
			err := stores.Alerts.Push(*alert)
			if err != nil {
				logger.Error("Failed to store alert",
					"error", err,
					"file_name", alert.FileName,
				)
			} else {
				logger.Info("Alert stored successfully",
					"file_name", alert.FileName,
					"rule_id", alert.RuleID,
				)
			}
		}
	} else {
		logger.Warn("Storage not initialized. Alerts will not be stored.")
	}

	// 如果有匹配项，返回命中结果
	if len(matches) > 0 {
		return &core.DetectionResult{
			DetectorName: d.name,
			Detected:     true,
			Matches:      matches,
		}, nil
	}

	// 返回未命中结果
	return &core.DetectionResult{
		DetectorName: d.name,
		Detected:     false,
		Matches:      []core.MatchDetail{},
	}, nil
}

// computeFileMD5 计算文件的 MD5 哈希值
func computeFileMD5(filePath string) (string, error) {
	// 以只读模式打开
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// 创建 MD5 hasher
	h := md5.New()

	// 流式拷贝，避免大文件占用过多内存
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	// 计算最终 hash
	hashBytes := h.Sum(nil)
	return hex.EncodeToString(hashBytes), nil
}
