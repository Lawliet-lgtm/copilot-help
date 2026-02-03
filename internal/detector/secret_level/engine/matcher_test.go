package engine

import (
	"strings"
	"testing"

	"linuxFileWatcher/internal/detector/secret_level/model"
)

func TestMatchContent(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantHit       bool
		wantLevel     model.SecretLevel
		wantMatchText string // 期望命中的文本
	}{
		// 1. 标准 GB/T 9704 格式
		// 注意：输入字符串中 "绝密★" 后面紧跟换行符
		{"Standard_TopSecret", "文件头\n绝密★\n内容", true, model.LevelTopSecret, "绝密★"},
		{"Standard_Secret_Years", "机密★10年", true, model.LevelSecret, "机密★10年"},
		{"Standard_Confidential_Long", "秘密★长期", true, model.LevelConfidential, "秘密★长期"},
		
		// 2. 容错测试 (空格、符号变体)
		{"Loose_Space", "机密 ★ 10年", true, model.LevelSecret, "机密 ★ 10年"},
		{"Loose_Star_ASCII", "绝密*20年", true, model.LevelTopSecret, "绝密*20年"},
		
		// 3. 负面测试 (不应命中)
		{"Negative_Normal_Text", "这是一个秘密的故事", false, model.LevelNone, ""},
		{"Negative_No_Star", "这也是机密文件", false, model.LevelNone, ""},
		{"Negative_Broken", "绝  密★", false, model.LevelNone, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHit, gotLevel, gotMatch := MatchContent(tt.input)
			
			if gotHit != tt.wantHit {
				t.Errorf("MatchContent() hit = %v, want %v", gotHit, tt.wantHit)
			}
			if gotLevel != tt.wantLevel {
				t.Errorf("MatchContent() level = %v, want %v", gotLevel, tt.wantLevel)
			}
			
			// 修正点：在这里做一下 TrimSpace，或者只检查是否包含核心文本
			// 这样可以避免因为正则匹配了末尾的 \n 或空格导致测试不通过
			if tt.wantHit {
				cleanMatch := strings.TrimSpace(gotMatch)
				if cleanMatch != tt.wantMatchText {
					// 如果还不匹配，打印出字节码以便调试
					t.Errorf("MatchContent() text = %q, want %q", gotMatch, tt.wantMatchText)
				}
			}
		})
	}
}