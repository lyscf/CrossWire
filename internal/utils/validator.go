package utils

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// Validator 验证器
type Validator struct{}

// NewValidator 创建验证器
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateNickname 验证昵称
func (v *Validator) ValidateNickname(nickname string) error {
	if len(nickname) < 2 || len(nickname) > 32 {
		return fmt.Errorf("nickname must be between 2 and 32 characters")
	}

	// 允许字母、数字、中文、下划线、连字符
	matched, _ := regexp.MatchString(`^[\p{L}\p{N}_-]+$`, nickname)
	if !matched {
		return fmt.Errorf("nickname contains invalid characters")
	}

	return nil
}

// ValidateChannelName 验证频道名称
func (v *Validator) ValidateChannelName(name string) error {
	if len(name) < 3 || len(name) > 64 {
		return fmt.Errorf("channel name must be between 3 and 64 characters")
	}

	// 允许字母、数字、中文、下划线、连字符、空格
	matched, _ := regexp.MatchString(`^[\p{L}\p{N}_\- ]+$`, name)
	if !matched {
		return fmt.Errorf("channel name contains invalid characters")
	}

	return nil
}

// ValidatePassword 验证密码强度
func (v *Validator) ValidatePassword(password string) error {
	if len(password) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}

	if len(password) > 128 {
		return fmt.Errorf("password too long (max 128 characters)")
	}

	return nil
}

// ValidateIP 验证IP地址
func (v *Validator) ValidateIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address")
	}
	return nil
}

// ValidatePort 验证端口号
func (v *Validator) ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

// ValidateMAC 验证MAC地址
func (v *Validator) ValidateMAC(mac string) error {
	_, err := net.ParseMAC(mac)
	if err != nil {
		return fmt.Errorf("invalid MAC address")
	}
	return nil
}

// ValidateFlagFormat 验证Flag格式
func (v *Validator) ValidateFlagFormat(flag string) error {
	if len(flag) == 0 {
		return fmt.Errorf("flag cannot be empty")
	}

	if len(flag) > 256 {
		return fmt.Errorf("flag too long (max 256 characters)")
	}

	return nil
}

// ValidateChallengeCategory 验证题目分类
func (v *Validator) ValidateChallengeCategory(category string) error {
	validCategories := []string{"Web", "Pwn", "Reverse", "Crypto", "Misc", "Forensics"}

	for _, valid := range validCategories {
		if category == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid category, must be one of: %s", strings.Join(validCategories, ", "))
}

// ValidateChallengeDifficulty 验证题目难度
func (v *Validator) ValidateChallengeDifficulty(difficulty string) error {
	validDifficulties := []string{"Easy", "Medium", "Hard", "Insane"}

	for _, valid := range validDifficulties {
		if difficulty == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid difficulty, must be one of: %s", strings.Join(validDifficulties, ", "))
}

// ValidateChallengePoints 验证题目分数
func (v *Validator) ValidateChallengePoints(points int) error {
	if points < 1 || points > 1000 {
		return fmt.Errorf("points must be between 1 and 1000")
	}
	return nil
}

// ValidateProgress 验证进度值
func (v *Validator) ValidateProgress(progress int) error {
	if progress < 0 || progress > 100 {
		return fmt.Errorf("progress must be between 0 and 100")
	}
	return nil
}

// ValidateRole 验证角色
func (v *Validator) ValidateRole(role string) error {
	validRoles := []string{"owner", "admin", "member", "readonly"}

	for _, valid := range validRoles {
		if role == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid role, must be one of: %s", strings.Join(validRoles, ", "))
}

// ValidateStatus 验证用户状态
func (v *Validator) ValidateStatus(status string) error {
	validStatuses := []string{"online", "busy", "away", "offline"}

	for _, valid := range validStatuses {
		if status == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid status, must be one of: %s", strings.Join(validStatuses, ", "))
}
