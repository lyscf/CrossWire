package storage

import (
	"time"

	"crosswire/internal/models"
)

// ChallengeRepository 题目数据仓库
type ChallengeRepository struct {
	db *Database
}

// NewChallengeRepository 创建题目仓库
func NewChallengeRepository(db *Database) *ChallengeRepository {
	return &ChallengeRepository{db: db}
}

// Create 创建题目
func (r *ChallengeRepository) Create(challenge *models.Challenge) error {
	return r.db.GetChannelDB().Create(challenge).Error
}

// GetByID 根据ID获取题目
func (r *ChallengeRepository) GetByID(challengeID string) (*models.Challenge, error) {
	var challenge models.Challenge
	err := r.db.GetChannelDB().Where("id = ?", challengeID).First(&challenge).Error
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

// GetByChannelID 获取频道所有题目
func (r *ChallengeRepository) GetByChannelID(channelID string) ([]*models.Challenge, error) {
	var challenges []*models.Challenge
	err := r.db.GetChannelDB().Where("channel_id = ?", channelID).
		Order(`
			CASE category 
				WHEN 'Web' THEN 1 
				WHEN 'Pwn' THEN 2 
				WHEN 'Reverse' THEN 3 
				WHEN 'Crypto' THEN 4 
				WHEN 'Misc' THEN 5 
				WHEN 'Forensics' THEN 6
				ELSE 7 
			END,
			points ASC
		`).
		Find(&challenges).Error
	if err != nil {
		return nil, err
	}
	return challenges, nil
}

// GetByCategory 按分类获取题目
func (r *ChallengeRepository) GetByCategory(channelID, category string) ([]*models.Challenge, error) {
	var challenges []*models.Challenge
	err := r.db.GetChannelDB().Where("channel_id = ? AND category = ?", channelID, category).
		Order("points ASC").
		Find(&challenges).Error
	if err != nil {
		return nil, err
	}
	return challenges, nil
}

// GetByStatus 按状态获取题目
func (r *ChallengeRepository) GetByStatus(channelID, status string) ([]*models.Challenge, error) {
	var challenges []*models.Challenge
	err := r.db.GetChannelDB().Where("channel_id = ? AND status = ?", channelID, status).
		Order("created_at DESC").
		Find(&challenges).Error
	if err != nil {
		return nil, err
	}
	return challenges, nil
}

// Update 更新题目
func (r *ChallengeRepository) Update(challenge *models.Challenge) error {
	return r.db.GetChannelDB().Save(challenge).Error
}

// Delete 删除题目
func (r *ChallengeRepository) Delete(challengeID string) error {
	return r.db.GetChannelDB().Where("id = ?", challengeID).Delete(&models.Challenge{}).Error
}

// 协作平台不需要验证Flag，直接明文存储（HashFlag/VerifyFlag 已移除）

// MarkAsSolved 标记题目为已解决
func (r *ChallengeRepository) MarkAsSolved(challengeID, solverID string) error {
	var challenge models.Challenge
	if err := r.db.GetChannelDB().Where("id = ?", challengeID).First(&challenge).Error; err != nil {
		return err
	}

	// 添加到solved_by列表
	solvedBy := append(challenge.SolvedBy, solverID)
	now := time.Now()

	return r.db.GetChannelDB().Model(&challenge).Updates(map[string]interface{}{
		"status":    "solved",
		"solved_by": solvedBy,
		"solved_at": now,
	}).Error
}

// AssignChallenge 分配题目给成员
func (r *ChallengeRepository) AssignChallenge(assignment *models.ChallengeAssignment) error {
	return r.db.GetChannelDB().Create(assignment).Error
}

// UnassignChallenge 取消分配
func (r *ChallengeRepository) UnassignChallenge(challengeID, memberID string) error {
	return r.db.GetChannelDB().Where("challenge_id = ? AND member_id = ?", challengeID, memberID).
		Delete(&models.ChallengeAssignment{}).Error
}

// GetAssignments 获取题目分配列表
func (r *ChallengeRepository) GetAssignments(challengeID string) ([]*models.ChallengeAssignment, error) {
	var assignments []*models.ChallengeAssignment
	err := r.db.GetChannelDB().Where("challenge_id = ?", challengeID).
		Find(&assignments).Error
	if err != nil {
		return nil, err
	}
	return assignments, nil
}

// GetMemberAssignments 获取成员被分配的所有题目
func (r *ChallengeRepository) GetMemberAssignments(memberID string) ([]*models.ChallengeAssignment, error) {
	var assignments []*models.ChallengeAssignment
	err := r.db.GetChannelDB().Where("member_id = ?", memberID).
		Find(&assignments).Error
	if err != nil {
		return nil, err
	}
	return assignments, nil
}

// UpdateProgress 更新题目进度
func (r *ChallengeRepository) UpdateProgress(progress *models.ChallengeProgress) error {
	// 检查是否已存在
	var existing models.ChallengeProgress
	err := r.db.GetChannelDB().Where("challenge_id = ? AND member_id = ?",
		progress.ChallengeID, progress.MemberID).
		Order("updated_at DESC").
		First(&existing).Error

	if err != nil {
		// 不存在，创建新记录
		return r.db.GetChannelDB().Create(progress).Error
	}

	// 存在，更新记录
	progress.ID = existing.ID
	return r.db.GetChannelDB().Save(progress).Error
}

// GetProgress 获取成员的题目进度
func (r *ChallengeRepository) GetProgress(challengeID, memberID string) (*models.ChallengeProgress, error) {
	var progress models.ChallengeProgress
	err := r.db.GetChannelDB().Where("challenge_id = ? AND member_id = ?", challengeID, memberID).
		Order("updated_at DESC").
		First(&progress).Error
	if err != nil {
		return nil, err
	}
	return &progress, nil
}

// GetTeamProgress 获取题目的团队进度
func (r *ChallengeRepository) GetTeamProgress(challengeID string) ([]*models.ChallengeProgress, error) {
	var progresses []*models.ChallengeProgress
	// 获取每个成员的最新进度
	err := r.db.GetChannelDB().Raw(`
		SELECT * FROM challenge_progress 
		WHERE challenge_id = ? 
		AND id IN (
			SELECT MAX(id) FROM challenge_progress 
			WHERE challenge_id = ? 
			GROUP BY member_id
		)
		ORDER BY updated_at DESC
	`, challengeID, challengeID).Scan(&progresses).Error

	if err != nil {
		return nil, err
	}
	return progresses, nil
}

// SubmitFlag 提交Flag
func (r *ChallengeRepository) SubmitFlag(submission *models.ChallengeSubmission) error {
	return r.db.GetChannelDB().Create(submission).Error
}

// GetSubmissions 获取题目的所有提交记录
func (r *ChallengeRepository) GetSubmissions(challengeID string) ([]*models.ChallengeSubmission, error) {
	var submissions []*models.ChallengeSubmission
	err := r.db.GetChannelDB().Where("challenge_id = ?", challengeID).
		Order("submitted_at DESC").
		Find(&submissions).Error
	if err != nil {
		return nil, err
	}
	return submissions, nil
}

// GetMemberSubmissions 获取成员的提交记录
func (r *ChallengeRepository) GetMemberSubmissions(memberID string) ([]*models.ChallengeSubmission, error) {
	var submissions []*models.ChallengeSubmission
	err := r.db.GetChannelDB().Where("member_id = ?", memberID).
		Order("submitted_at DESC").
		Find(&submissions).Error
	if err != nil {
		return nil, err
	}
	return submissions, nil
}

// AddHint 添加提示
func (r *ChallengeRepository) AddHint(hint *models.ChallengeHint) error {
	return r.db.GetChannelDB().Create(hint).Error
}

// GetHints 获取题目的所有提示
func (r *ChallengeRepository) GetHints(challengeID string) ([]*models.ChallengeHint, error) {
	var hints []*models.ChallengeHint
	err := r.db.GetChannelDB().Where("challenge_id = ?", challengeID).
		Order("order_num ASC").
		Find(&hints).Error
	if err != nil {
		return nil, err
	}
	return hints, nil
}

// UnlockHint 解锁提示
func (r *ChallengeRepository) UnlockHint(hintID, memberID string) error {
	var hint models.ChallengeHint
	if err := r.db.GetChannelDB().Where("id = ?", hintID).First(&hint).Error; err != nil {
		return err
	}

	// 添加到unlocked_by列表
	unlockedBy := append(hint.UnlockedBy, memberID)

	return r.db.GetChannelDB().Model(&hint).Update("unlocked_by", unlockedBy).Error
}

// IsHintUnlocked 检查提示是否已解锁
func (r *ChallengeRepository) IsHintUnlocked(hintID, memberID string) (bool, error) {
	var hint models.ChallengeHint
	if err := r.db.GetChannelDB().Where("id = ?", hintID).First(&hint).Error; err != nil {
		return false, err
	}

	for _, uid := range hint.UnlockedBy {
		if uid == memberID {
			return true, nil
		}
	}
	return false, nil
}

// GetChallengeStats 获取题目统计信息
func (r *ChallengeRepository) GetChallengeStats(channelID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总题目数
	var totalCount int64
	r.db.GetChannelDB().Model(&models.Challenge{}).
		Where("channel_id = ?", channelID).
		Count(&totalCount)
	stats["total"] = totalCount

	// 按状态统计
	var statusCounts []struct {
		Status string
		Count  int64
	}
	r.db.GetChannelDB().Model(&models.Challenge{}).
		Where("channel_id = ?", channelID).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts)

	statusMap := make(map[string]int64)
	for _, sc := range statusCounts {
		statusMap[sc.Status] = sc.Count
	}
	stats["by_status"] = statusMap

	// 按分类统计
	var categoryCounts []struct {
		Category string
		Count    int64
	}
	r.db.GetChannelDB().Model(&models.Challenge{}).
		Where("channel_id = ?", channelID).
		Select("category, COUNT(*) as count").
		Group("category").
		Scan(&categoryCounts)

	categoryMap := make(map[string]int64)
	for _, cc := range categoryCounts {
		categoryMap[cc.Category] = cc.Count
	}
	stats["by_category"] = categoryMap

	return stats, nil
}

// CountAssignedToMember 统计分配给成员的题目数（协作平台）
func (r *ChallengeRepository) CountAssignedToMember(memberID string) (int, error) {
	var count int64

	err := r.db.GetChannelDB().
		Model(&models.ChallengeAssignment{}).
		Where("member_id = ?", memberID).
		Count(&count).Error

	return int(count), err
}

// GetMemberContributionStats 获取成员贡献统计（协作平台：参与题目数）
func (r *ChallengeRepository) GetMemberContributionStats(memberID string) (int, error) {
	// 简化版：统计分配给该成员的题目数
	return r.CountAssignedToMember(memberID)
}

// GetAllMembersContributionStats 批量获取所有成员的贡献统计（性能优化）
func (r *ChallengeRepository) GetAllMembersContributionStats() (map[string]int, error) {
	// 一次查询获取所有成员的参与题目数
	var results []struct {
		MemberID string
		Count    int64
	}

	err := r.db.GetChannelDB().
		Model(&models.ChallengeAssignment{}).
		Select("member_id, COUNT(*) as count").
		Group("member_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 转换为map
	statsMap := make(map[string]int)
	for _, r := range results {
		statsMap[r.MemberID] = int(r.Count)
	}

	return statsMap, nil
}

// TODO: 实现以下方法
// - SearchChallenges() 搜索题目
// - GetChallengeLeaderboard() 获取解题排行榜（如需要）
