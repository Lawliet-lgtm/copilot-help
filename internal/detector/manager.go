package detector

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"linuxFileWatcher/internal/model"
	"linuxFileWatcher/internal/detector/govcheck"
	"linuxFileWatcher/internal/detector/secret_level"
)

// SubDetector 定义所有子检测模块必须实现的通用接口
type SubDetector interface {
	DetectFile(ctx context.Context, filePath string) (*model.SubDetectResult, error)
}

// GlobalConfig 全局检测配置
type GlobalConfig struct {
	EnableElectronicLabel bool
	EnableSecretMarker    bool
	EnableLayout          bool
	EnableHash            bool
	EnableKeywords        bool

	SecretMarkerOCR bool

	// 公文版式检测配置
	LayoutThreshold float64
	LayoutEnableOCR bool

	// 基础信息
	CurrentCompany      string
	CurrentComputerName string
	CurrentOrgID        string
	CurrentOrgPath      string
	CurrentUserName     string
	CurrentUserID       string
}

// Manager 涉密信息检测模块管理器
type Manager struct {
	config GlobalConfig

	secretMarkerDetector    secret_level.Detector
	electronicLabelDetector SubDetector
	layoutDetector          govcheck.Detector // 公文版式检测器
	hashDetector            SubDetector
	keywordsDetector        SubDetector
}

// NewManager 初始化管理器
func NewManager(cfg GlobalConfig) *Manager {
	mgr := &Manager{
		config: cfg,
	}

	// 1. 初始化密级标志检测器
	markerCfg := secret_level.Config{
		EnableOCR: cfg.SecretMarkerOCR,
	}
	mgr.secretMarkerDetector = secret_level.NewDetector(markerCfg)

	// 2. 初始化公文版式检测器
	layoutCfg := govcheck.DefaultConfig()
	if cfg.LayoutThreshold > 0 {
		layoutCfg.Threshold = cfg.LayoutThreshold
	}
	layoutCfg.EnableOCR = cfg.LayoutEnableOCR
	mgr.layoutDetector = govcheck.NewDetector(layoutCfg)

	// 3. 其他子模块初始化...
	// mgr.electronicLabelDetector = ...
	// mgr.hashDetector = ...
	// mgr.keywordsDetector = ...

	return mgr
}

// UpdateConfig 更新配置
func (m *Manager) UpdateConfig(newCfg GlobalConfig) {
	m.config = newCfg

	// 更新公文版式检测器配置（如果需要热更新）
	// 注意：当前实现需要重新创建检测器才能更新配置
}

// Detect 主检测入口
func (m *Manager) Detect(ctx context.Context, filePath string) (bool, *model.AlertRecord, *model.AlertLogItem, error) {
	// 0. 预处理：获取文件通用信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, nil, nil, err
	}

	fileMD5, err := calculateMD5(filePath)
	if err != nil {
		fileMD5 = ""
	}

	// 构造结果处理闭包
	handleResult := func(res *model.SubDetectResult) (bool, *model.AlertRecord, *model.AlertLogItem, error) {
		if res == nil || !res.IsSecret {
			return false, nil, nil, nil
		}

		record := &model.AlertRecord{
			ID:            generateAlertID(),
			Time:          time.Now().Format("2006-01-02 15:04:05"),
			RuleID:        res.RuleID,
			RuleDesc:      res.RuleDesc,
			FilterType:    1,
			FileSummary:   "",
			AlertType:     model.AlertType(res.AlertType),
			FileMD5:       fileMD5,
			FilePath:      filePath,
			FileName:      fileInfo.Name(),
			FileSize:      int(fileInfo.Size()),
			HighlightText: res.MatchedText,
			FileDesc:      res.ContextText,
			Company:       m.config.CurrentCompany,
			ComputerName:  m.config.CurrentComputerName,
			OrgID:         m.config.CurrentOrgID,
			OrgPath:       m.config.CurrentOrgPath,
			UserName:      m.config.CurrentUserName,
			UserID:        m.config.CurrentUserID,
			FileLevel:     int(res.SecretLevel),
		}

		logItem := &model.AlertLogItem{
			FileName: fileInfo.Name(),
			FilePath: filePath,
			FileMD5:  fileMD5,
			Time:     record.Time,
		}

		return true, record, logItem, nil
	}

	// 1. 电子密级检测
	if m.config.EnableElectronicLabel && m.electronicLabelDetector != nil {
		res, err := m.electronicLabelDetector.DetectFile(ctx, filePath)
		if err == nil && res != nil && res.IsSecret {
			return handleResult(res)
		}
	}

	// 2. 密级标志检测
	if m.config.EnableSecretMarker && m.secretMarkerDetector != nil {
		res, err := m.secretMarkerDetector.DetectFile(ctx, filePath)
		if err == nil && res != nil && res.IsSecret {
			return handleResult(res)
		}
	}

	// 3. 公文版式检测
	if m.config.EnableLayout && m.layoutDetector != nil {
		res, err := m.layoutDetector.DetectFile(ctx, filePath)
		if err == nil && res != nil && res.IsSecret {
			return handleResult(res)
		}
	}

	// 4. 哈希检测
	if m.config.EnableHash && m.hashDetector != nil {
		res, err := m.hashDetector.DetectFile(ctx, filePath)
		if err == nil && res != nil && res.IsSecret {
			return handleResult(res)
		}
	}

	// 5. 关键词检测
	if m.config.EnableKeywords && m.keywordsDetector != nil {
		res, err := m.keywordsDetector.DetectFile(ctx, filePath)
		if err == nil && res != nil && res.IsSecret {
			return handleResult(res)
		}
	}

	return false, nil, nil, nil
}

// calculateMD5 计算文件 MD5
func calculateMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// generateAlertID 生成告警 ID
func generateAlertID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}