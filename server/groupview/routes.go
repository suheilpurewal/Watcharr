package groupview

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(db *gorm.DB, api *gin.RouterGroup, authRequired gin.HandlerFunc) {
	a := &API{DB: db}
	g := api.Group("/group")
	
	// Public routes (no auth required)
	g.GET("/members", a.GetMembers)
	g.POST("/members", a.PostMember)
	g.POST("/viewings", a.PostViewing)
	g.GET("/history", a.GetHistory)
	
	// Authenticated routes
	g.GET("/my-pending-ratings", authRequired, a.GetMyPendingRatings)
	g.PUT("/attendance/:id/rating", authRequired, a.UpdateAttendanceRating)
	
	// Group management routes
	g.GET("/info", authRequired, a.GetGroupInfo)
	g.PUT("/name", authRequired, a.UpdateGroupName)
	
	// Phase 2: Family history and content sharing routes
	g.GET("/family-history", authRequired, a.GetFamilyHistory)
	g.POST("/share-content", authRequired, a.ShareContentToFamily)
	
	// Debug/utility routes
	g.GET("/ensure-user-in-group", authRequired, a.EnsureUserInGroup)
	g.GET("/debug-status", authRequired, a.DebugGroupStatus)
}
