package dumate

import "testing"

func TestBuildPointsUsesSoonestActivePackOnly(t *testing.T) {
	soon, later := int64(1893456000), int64(1924992000)
	overview := Overview{TotalPoints: 100, UsedPoints: 60, Packs: []Pack{
		{Status: "expired", TotalPoints: 999, ExpireDate: &soon},
		{Status: "active", TotalPoints: 30, ExpireDate: &later},
		{Status: "active", TotalPoints: 10, ExpireDate: &soon},
		{Status: "active", TotalPoints: 500},
	}}
	points := BuildPoints(overview, LoginBonus{HasIssued: true, TotalTimes: 3}, true)
	if points.RemainingPoints != 40 || points.TotalPoints != 100 || points.UsedPoints != 60 {
		t.Fatalf("points = %+v", points)
	}
	if points.ExpiringAt == "" || points.ExpiringPoints != 10 {
		t.Fatalf("expiry = (%q, %v)", points.ExpiringAt, points.ExpiringPoints)
	}
	if !points.CheckinIssued || points.CheckinDays != 3 || !points.CheckinKnown {
		t.Fatalf("checkin = %+v", points)
	}
}

func TestBuildPointsClampsAndHidesUnknownCheckin(t *testing.T) {
	points := BuildPoints(Overview{TotalPoints: 5, UsedPoints: 9}, LoginBonus{HasIssued: true, TotalTimes: 4}, false)
	if points.RemainingPoints != 0 || points.ExpiringPoints != 0 || points.ExpiringAt != "" {
		t.Fatalf("points = %+v", points)
	}
	if points.CheckinKnown || points.CheckinIssued || points.CheckinDays != 0 {
		t.Fatalf("unknown checkin leaked: %+v", points)
	}
}

func TestBuildPointsRoundsAndIgnoresInvalidPackExpiry(t *testing.T) {
	zero, negative := int64(0), int64(-1)
	points := BuildPoints(Overview{TotalPoints: 1.239, UsedPoints: 0.001, Packs: []Pack{
		{Status: "active", TotalPoints: 1, ExpireDate: &zero},
		{Status: "active", TotalPoints: 1, ExpireDate: &negative},
	}}, LoginBonus{}, false)
	if points.RemainingPoints != 1.24 || points.TotalPoints != 1.24 || points.ExpiringAt != "" {
		t.Fatalf("points = %+v", points)
	}
}
