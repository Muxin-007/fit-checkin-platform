package response

type File struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	AuditStatus string `json:"auditStatus"`
}

type User struct {
	ID              string `json:"id"`
	Nickname        string `json:"nickname"`
	Avatar          *File  `json:"avatar,omitempty"`
	ReminderEnabled bool   `json:"reminderEnabled"`
	WeightPublic    bool   `json:"weightPublic"`
	Status          string `json:"status"`
}

type Login struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
	User      User   `json:"user"`
}

type Stats struct {
	CurrentStreak int `json:"currentStreak"`
	LongestStreak int `json:"longestStreak"`
	MonthCount    int `json:"monthCount"`
	TotalCount    int `json:"totalCount"`
	TotalMinutes  int `json:"totalMinutes"`
}

type GroupSummary struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Avatar           *File  `json:"avatar,omitempty"`
	Description      string `json:"description"`
	MemberCount      int    `json:"memberCount"`
	CheckedCount     int    `json:"checkedCount"`
	CurrentChecked   bool   `json:"currentChecked"`
	Role             string `json:"role"`
	MembershipStatus string `json:"membershipStatus"`
}

type Activity struct {
	CheckinID       string `json:"checkinId"`
	UserID          string `json:"userId"`
	Nickname        string `json:"nickname"`
	Avatar          *File  `json:"avatar,omitempty"`
	ExerciseType    string `json:"exerciseType"`
	DurationMinutes int    `json:"durationMinutes"`
	CheckinAt       int64  `json:"checkinAt"`
}

type Home struct {
	TodayChecked bool           `json:"todayChecked"`
	Stats        Stats          `json:"stats"`
	Groups       []GroupSummary `json:"groups"`
	Activities   []Activity     `json:"activities"`
}

type Invitation struct {
	Code            string `json:"code"`
	ExpiresAt       int64  `json:"expiresAt"`
	GroupID         string `json:"groupId"`
	GroupName       string `json:"groupName"`
	GroupAvatar     *File  `json:"groupAvatar,omitempty"`
	MemberCount     int    `json:"memberCount"`
	WeeklyTarget    int    `json:"weeklyTarget"`
	RequireApproval bool   `json:"requireApproval"`
}

type MemberStatus struct {
	MemberID        string `json:"memberId"`
	UserID          string `json:"userId"`
	Nickname        string `json:"nickname"`
	Avatar          *File  `json:"avatar,omitempty"`
	Role            string `json:"role"`
	Status          string `json:"status"`
	Checked         bool   `json:"checked"`
	AuditPending    bool   `json:"auditPending"`
	CheckinID       string `json:"checkinId,omitempty"`
	CheckinAt       int64  `json:"checkinAt,omitempty"`
	ExerciseType    string `json:"exerciseType,omitempty"`
	DurationMinutes int    `json:"durationMinutes,omitempty"`
	Image           *File  `json:"image,omitempty"`
	CurrentStreak   int    `json:"currentStreak"`
}

type GroupDetail struct {
	GroupSummary
	Announcement     string         `json:"announcement"`
	WeeklyTarget     int            `json:"weeklyTarget"`
	ReminderTime     string         `json:"reminderTime"`
	RequireApproval  bool           `json:"requireApproval"`
	MemberLimit      int            `json:"memberLimit"`
	CheckedMembers   []MemberStatus `json:"checkedMembers"`
	PendingMembers   []MemberStatus `json:"pendingMembers"`
	UncheckedMembers []MemberStatus `json:"uncheckedMembers"`
}

type Checkin struct {
	ID                string         `json:"id"`
	Date              string         `json:"date"`
	ExerciseType      string         `json:"exerciseType"`
	DurationMinutes   int            `json:"durationMinutes"`
	Content           string         `json:"content"`
	Calories          *int           `json:"calories,omitempty"`
	Weight            *float64       `json:"weight,omitempty"`
	WeightPublic      bool           `json:"weightPublic"`
	Mood              string         `json:"mood"`
	AuditStatus       string         `json:"auditStatus"`
	AuditDetail       string         `json:"auditDetail,omitempty"`
	CanManage         bool           `json:"canManage"`
	CreatedAt         int64          `json:"createdAt"`
	Images            []File         `json:"images"`
	ImageAuditSummary map[string]int `json:"imageAuditSummary,omitempty"`
	GroupIDs          []string       `json:"groupIds"`
	GroupNames        []string       `json:"groupNames"`
}

type Calendar struct {
	Month string    `json:"month"`
	Stats Stats     `json:"stats"`
	Items []Checkin `json:"items"`
}

type Page[T any] struct {
	List     []T `json:"list"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}
