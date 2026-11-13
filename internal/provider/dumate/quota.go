package dumate

import (
	"math"
	"time"
)

// Points 是公开的积分视图；所有字段都可展示，不包含 Cookie 或 token。
type Points struct {
	// CheckinIssued 与 CheckinDays 来自登录奖励接口；CheckinKnown 为 false 时
	// 这两个字段没有意义，界面不能显示成「今天未签到」。
	CheckinIssued   bool    `json:"checkinIssued"`
	CheckinDays     int     `json:"checkinDays"`
	TotalPoints     float64 `json:"totalPoints"`
	UsedPoints      float64 `json:"usedPoints"`
	RemainingPoints float64 `json:"remainingPoints"`
	// ExpiringAt 是最近一个有效积分包的到期时间；ExpiringPoints 是该包中仍可能
	// 剩余的积分下界，不能把积分包原始总额当作确定剩余额。
	ExpiringAt     string  `json:"expiringAt,omitempty"`
	ExpiringPoints float64 `json:"expiringPoints,omitempty"`
	CheckinKnown   bool    `json:"checkinKnown"`
}

// BuildPoints 合并总览与签到状态；时间使用 UTC RFC3339，前端按本地时区显示。
func BuildPoints(overview Overview, bonus LoginBonus, bonusKnown bool) Points {
	remaining := math.Max(0, overview.TotalPoints-overview.UsedPoints)
	points := Points{
		TotalPoints:     roundPoints(overview.TotalPoints),
		UsedPoints:      roundPoints(overview.UsedPoints),
		RemainingPoints: roundPoints(remaining),
		CheckinKnown:    bonusKnown,
	}
	if bonusKnown {
		points.CheckinIssued, points.CheckinDays = bonus.HasIssued, bonus.TotalTimes
	}
	var soonest *Pack
	for i := range overview.Packs {
		pack := overview.Packs[i]
		if pack.Status != "active" || pack.ExpireDate == nil || *pack.ExpireDate <= 0 {
			continue
		}
		if soonest == nil || *pack.ExpireDate < *soonest.ExpireDate {
			soonest = &overview.Packs[i]
		}
	}
	if soonest != nil {
		points.ExpiringAt = time.Unix(*soonest.ExpireDate, 0).UTC().Format(time.RFC3339)
		points.ExpiringPoints = roundPoints(math.Min(remaining, soonest.TotalPoints))
	}
	return points
}

func roundPoints(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Round(value*100) / 100
}
