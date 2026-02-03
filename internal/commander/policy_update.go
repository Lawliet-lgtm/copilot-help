package commander

import (
	"encoding/json"
	"linuxFileWatcher/internal/logger"
	"linuxFileWatcher/internal/model"
	"os"
	"sort"
)

// 处理关键词策略
func (d *SimpleDispatcher) processKeywordUpdate(op string, newRules []model.KeywordDetectRule) {
	filePath := "keyword_policy.json"

	// A. 读取本地文件内容
	var config model.KeywordDetectConfig
	if content, err := os.ReadFile(filePath); err == nil {
		json.Unmarshal(content, &config)
	}

	// B. 根据 Cmd 判断操作类型
	switch op {
	case model.PolicyCmdReset:
		// 全量覆盖：直接替换
		config.Rules = newRules

	case model.PolicyCmdAdd:
		// 增量添加：合并 + 去重
		// 使用 map 以 RuleID 为 Key 进行合并
		ruleMap := make(map[int64]model.KeywordDetectRule)
		for _, r := range config.Rules {
			ruleMap[r.RuleID] = r
		}
		for _, r := range newRules {
			ruleMap[r.RuleID] = r
		} // 新规则覆盖旧的（如果有 ID 冲突）

		config.Rules = make([]model.KeywordDetectRule, 0)
		for _, r := range ruleMap {
			config.Rules = append(config.Rules, r)
		}

	case model.PolicyCmdDel:
		// 增量删除：剔除
		ruleMap := make(map[int64]model.KeywordDetectRule)
		for _, r := range config.Rules {
			ruleMap[r.RuleID] = r
		}
		for _, r := range newRules {
			delete(ruleMap, r.RuleID)
		}

		config.Rules = make([]model.KeywordDetectRule, 0)
		for _, r := range ruleMap {
			config.Rules = append(config.Rules, r)
		}
	}

	sort.Slice(config.Rules, func(i, j int) bool {
		return config.Rules[i].RuleID < config.Rules[j].RuleID
	})
	// C. 写回本地
	d.writeJSON(filePath, config)
}

// 处理哈希策略
func (d *SimpleDispatcher) processHashUpdate(op string, newRules []model.HashDetectRule) {
	filePath := "hash_policy.json"

	// 1. 读取本地现有哈希策略
	var config model.HashDetectConfig
	if content, err := os.ReadFile(filePath); err == nil {
		json.Unmarshal(content, &config)
	}

	// 2. 根据操作类型处理
	switch op {
	case model.PolicyCmdReset: // "reset"
		// 全量覆盖
		config.Rules = newRules

	case model.PolicyCmdAdd: // "add"
		// 增量添加：使用 RuleID 做 Key 来合并和去重
		ruleMap := make(map[int64]model.HashDetectRule)
		// 把旧的装进 Map
		for _, r := range config.Rules {
			ruleMap[r.RuleID] = r
		}
		// 把新的装进 Map，ID 冲突时以新的为准
		for _, r := range newRules {
			ruleMap[r.RuleID] = r
		}

		// 重新转回切片
		config.Rules = make([]model.HashDetectRule, 0)
		for _, r := range ruleMap {
			config.Rules = append(config.Rules, r)
		}

	case model.PolicyCmdDel: // "del"
		// 增量删除
		ruleMap := make(map[int64]model.HashDetectRule)
		for _, r := range config.Rules {
			ruleMap[r.RuleID] = r
		}
		// 从 Map 中移除服务器指定的 ID
		for _, r := range newRules {
			delete(ruleMap, r.RuleID)
		}

		config.Rules = make([]model.HashDetectRule, 0)
		for _, r := range ruleMap {
			config.Rules = append(config.Rules, r)
		}
	}

	sort.Slice(config.Rules, func(i, j int) bool {
		return config.Rules[i].RuleID < config.Rules[j].RuleID
	})

	// 3. 写回本地
	d.writeJSON(filePath, config)
}

func (d *SimpleDispatcher) writeJSON(path string, data interface{}) {
	// 1. 定义临时文件
	tmpPath := path + ".tmp"

	// 2. 创建并写入数据
	// os.Create 会默认创建权限为 0666 的文件（受 umask 影响，通常所有人可读写）
	file, err := os.Create(tmpPath)
	if err != nil {
		logger.Error("无法创建文件", "path", tmpPath, "err", err)
		return
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		file.Close()
		logger.Error("JSON 编码失败", "err", err)
		return
	}
	_ = file.Sync()
	file.Close()

	// 3. 原子替换
	// 在 Linux 上，这步操作非常快，且能保证 path 文件永远是完整的
	if err := os.Rename(tmpPath, path); err != nil {
		logger.Error("替换文件失败", "err", err)
		return
	}

}
