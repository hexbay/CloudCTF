package bootstrap

import (
	"time"

	"cloudctf/backend/internal/models"

	"gorm.io/gorm"
)

func seedNotificationData(db *gorm.DB) error {
	var annCount int64
	if err := db.Model(&models.Announcement{}).Count(&annCount).Error; err != nil {
		return err
	}
	if annCount == 0 {
		now := time.Now()
		announcement := models.Announcement{
			Title:       "欢迎使用 CloudCTF",
			Content:     "<h2>欢迎使用 CloudCTF CTF 平台</h2><p>这是一条系统公告，欢迎各位参赛者。</p>",
			Summary:     "欢迎使用 CloudCTF",
			AuthorID:    1,
			IsPinned:    true,
			PinOrder:    1,
			IsPublished: true,
			TargetType:  "all",
			PublishedAt: &now,
			ViewCount:   10,
		}
		if err := db.Create(&announcement).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedTheoryExamData(db *gorm.DB) error {
	var theoryCount int64
	if err := db.Model(&models.TheoryQuestion{}).Count(&theoryCount).Error; err != nil {
		return err
	}
	if theoryCount == 0 {
		questions := []models.TheoryQuestion{
			{Question: "以下哪些是常见的 Web 安全漏洞？", Points: 5, Category: "Web安全", Difficulty: 1, QuestionType: "multiple", Context: `{"options":[{"text":"SQL注入","checked":true},{"text":"XSS","checked":true},{"text":"CSRF","checked":true},{"text":"缓冲区溢出","checked":false}]}`},
			{Question: "HTTPS 使用的加密协议是 TLS", Points: 2, Category: "网络安全", Difficulty: 1, QuestionType: "judge", Context: `{"answer":true}`},
			{Question: "CTF 中 PWN 方向主要考察什么？", Points: 10, Category: "基础", Difficulty: 2, QuestionType: "text", Context: `{"answer":"二进制漏洞利用与逆向工程"}`},
			{Question: "以下哪个是公认最安全的加密算法？", Points: 5, Category: "密码学", Difficulty: 1, QuestionType: "single", Context: `{"options":[{"text":"AES-256","checked":true},{"text":"DES","checked":false},{"text":"MD5","checked":false},{"text":"Base64","checked":false}]}`},
			{Question: "OWASP Top 10 2021 中排名第一的风险是什么？", Points: 5, Category: "Web安全", Difficulty: 1, QuestionType: "single", Context: `{"options":[{"text":"Broken Access Control","checked":true},{"text":"Cryptographic Failures","checked":false},{"text":"Injection","checked":false},{"text":"Security Misconfiguration","checked":false}]}`},
		}
		for _, q := range questions {
			if err := db.Create(&q).Error; err != nil {
				return err
			}
		}
	}

	var examCount int64
	if err := db.Model(&models.Exam{}).Count(&examCount).Error; err != nil {
		return err
	}
	if examCount == 0 {
		exam := models.Exam{
			CompetitionID: nil,
			Title:         "安全基础知识测试",
			TotalScore:    100,
			IsActive:      true,
			Context:       `{"questions":[{"id":1,"question":"以下哪些是常见的 Web 安全漏洞？","points":5,"category":"Web安全","difficulty":1,"question_type":"multiple","context":{"options":[{"text":"SQL注入","checked":true},{"text":"XSS","checked":true},{"text":"CSRF","checked":true},{"text":"缓冲区溢出","checked":false}]}},{"id":2,"question":"HTTPS 使用的加密协议是 TLS","points":2,"category":"网络安全","difficulty":1,"question_type":"judge","context":{"answer":true}},{"id":3,"question":"CTF 中 PWN 方向主要考察什么？","points":10,"category":"基础","difficulty":2,"question_type":"text","context":{"answer":"二进制漏洞利用与逆向工程"}},{"id":4,"question":"以下哪个是公认最安全的加密算法？","points":5,"category":"密码学","difficulty":1,"question_type":"single","context":{"options":[{"text":"AES-256","checked":true},{"text":"DES","checked":false},{"text":"MD5","checked":false},{"text":"Base64","checked":false}]}},{"id":5,"question":"OWASP Top 10 2021 中排名第一的风险是什么？","points":5,"category":"Web安全","difficulty":1,"question_type":"single","context":{"options":[{"text":"Broken Access Control","checked":true},{"text":"Cryptographic Failures","checked":false},{"text":"Injection","checked":false},{"text":"Security Misconfiguration","checked":false}]}}],"time_limit":3600}`,
		}
		if err := db.Create(&exam).Error; err != nil {
			return err
		}
	}
	return nil
}
