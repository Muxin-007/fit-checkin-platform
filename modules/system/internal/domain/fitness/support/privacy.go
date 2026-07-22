package support

const DefaultGroupName = "日常运动小组"

var groupNames = map[string]struct{}{
	"日常运动小组": {},
	"跑步打卡小组": {},
	"健身训练小组": {},
	"步行打卡小组": {},
	"综合运动小组": {},
}

func PublicGroupName(name string) string {
	if _, ok := groupNames[name]; ok {
		return name
	}
	return DefaultGroupName
}
