package request

type WechatLogin struct {
	Code string `json:"code" binding:"required"`
}

type UpdateProfile struct {
	Nickname     *string `json:"nickname" binding:"omitempty,max=30"`
	AvatarFileID *string `json:"avatarFileId"`
}

type UpdateSettings struct {
	ReminderEnabled *bool `json:"reminderEnabled"`
}

type CreateGroup struct {
	Name            string `json:"name" binding:"required,oneof=日常运动小组 跑步打卡小组 健身训练小组 步行打卡小组 综合运动小组"`
	WeeklyTarget    int    `json:"weeklyTarget" binding:"required,min=1,max=7"`
	ReminderTime    string `json:"reminderTime" binding:"required,len=5"`
	RequireApproval bool   `json:"requireApproval"`
	MemberLimit     int    `json:"memberLimit" binding:"required,min=2,max=200"`
}

type UpdateGroup struct {
	ID              string  `json:"id" binding:"required"`
	Name            *string `json:"name" binding:"omitempty,oneof=日常运动小组 跑步打卡小组 健身训练小组 步行打卡小组 综合运动小组"`
	WeeklyTarget    *int    `json:"weeklyTarget" binding:"omitempty,min=1,max=7"`
	ReminderTime    *string `json:"reminderTime" binding:"omitempty,len=5"`
	RequireApproval *bool   `json:"requireApproval"`
	MemberLimit     *int    `json:"memberLimit" binding:"omitempty,min=2,max=200"`
}

type CreateInvitation struct {
	GroupID    string `json:"groupId" binding:"required"`
	ValidHours int    `json:"validHours" binding:"omitempty,min=1,max=720"`
}

type JoinGroup struct {
	Code string `json:"code" binding:"required"`
}

type ReviewMember struct {
	MemberID string `json:"memberId" binding:"required"`
	Approve  bool   `json:"approve"`
}

type UpsertCheckin struct {
	ExerciseType    string   `json:"exerciseType" binding:"required,oneof=running walking cycling strength swimming yoga ball rope other"`
	DurationMinutes int      `json:"durationMinutes" binding:"required,min=1,max=1440"`
	GroupIDs        []string `json:"groupIds" binding:"max=50"`
	ImageFileIDs    []string `json:"imageFileIds" binding:"max=9"`
}

type AuthorizeSubscription struct {
	TemplateID string `json:"templateId" binding:"required"`
	Accepted   bool   `json:"accepted"`
}

type ManualReminder struct {
	GroupID        string `json:"groupId" binding:"required"`
	TargetMemberID string `json:"targetMemberId"`
	AllPending     bool   `json:"allPending"`
}

type AdminPage struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Keyword  string `form:"keyword"`
	Status   string `form:"status"`
}

type AdminUserStatus struct {
	Status string `json:"status" binding:"required,oneof=active disabled"`
}

type AdminCheckinAudit struct {
	Status string `json:"status" binding:"required,oneof=approved rejected"`
	Detail string `json:"detail" binding:"max=300"`
}

type UpdateConfig struct {
	Value string `json:"value" binding:"max=20000"`
}
