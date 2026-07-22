package fitness

import (
	"github.com/gin-gonic/gin"

	"platform/modules/system/internal/domain/fitness/support"
)

func InitRouter(router *gin.RouterGroup) {
	api := &API{Service: &Service{}}

	router.POST("auth/login", api.Login)
	router.GET("config", api.PublicConfig)
	router.GET("files/:id", api.DownloadFile)
	router.GET("security/media-callback", api.VerifyMediaCallback)
	router.POST("security/media-callback", api.MediaCallback)
	router.GET("ops", api.AdminWeb)
	router.GET("ops/*asset", api.AdminWeb)

	auth := router.Group("")
	auth.Use(support.Auth())
	{
		auth.POST("auth/logout", api.Logout)
		auth.GET("home", api.Home)
		auth.GET("profile", api.Profile)
		auth.PUT("profile", api.UpdateProfile)
		auth.PUT("profile/settings", api.UpdateSettings)
		auth.DELETE("profile", api.CancelAccount)
		auth.POST("files/images", api.UploadImage)
		auth.GET("files/:id/status", api.FileStatus)

		auth.GET("groups", api.ListGroups)
		auth.POST("groups", api.CreateGroup)
		auth.GET("groups/:id", api.GetGroup)
		auth.PUT("groups/:id", api.UpdateGroup)
		auth.DELETE("groups/:id", api.DissolveGroup)
		auth.GET("groups/:id/invitation", api.GetInvitation)
		auth.POST("groups/:id/invitations", api.CreateInvitation)
		auth.POST("groups/:id/members/review", api.ReviewMember)
		auth.DELETE("groups/:id/members/me", api.LeaveGroup)
		auth.DELETE("groups/:id/members/:memberId", api.RemoveMember)
		auth.GET("invitations/:code", api.InvitationPreview)
		auth.POST("invitations/join", api.JoinGroup)

		auth.GET("checkins/today", api.TodayCheckin)
		auth.PUT("checkins/today", api.UpsertCheckin)
		auth.GET("checkins/calendar", api.Calendar)
		auth.GET("checkins/:id", api.GetCheckin)
		auth.DELETE("checkins/:id", api.DeleteCheckin)

		auth.POST("subscriptions", api.AuthorizeSubscription)
		auth.POST("reminders", api.ManualReminder)
	}

	admin := router.Group("admin")
	admin.Use(support.AdminAuth())
	{
		admin.GET("dashboard", api.AdminDashboard)
		admin.GET("users", api.AdminUsers)
		admin.PUT("users/:id/status", api.AdminUpdateUserStatus)
		admin.GET("groups", api.AdminGroups)
		admin.DELETE("groups/:id", api.AdminDissolveGroup)
		admin.GET("checkins", api.AdminCheckins)
		admin.PUT("checkins/:id/audit", api.AdminAuditCheckin)
		admin.DELETE("checkins/:id", api.AdminDeleteCheckin)
		admin.GET("reminders", api.AdminReminders)
		admin.PUT("configs/:key", api.AdminUpdateConfig)
	}
}

func StartReminderScheduler() {
	support.StartScheduler()
}
