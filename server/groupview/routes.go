package groupview

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(db *gorm.DB, api *gin.RouterGroup) {
	a := &API{DB: db}
	g := api.Group("/group")
	
	// Import AuthRequired from main package
	// Note: This assumes AuthRequired is available in the main package
	// If not, we'll need to import it properly
	
	g.GET("/members", a.GetMembers)
	g.POST("/members", a.PostMember)
	g.POST("/viewings", a.PostViewing)
	g.GET("/history", a.GetHistory)
	
	// These routes require authentication
	g.GET("/my-pending-ratings", a.GetMyPendingRatings)
	g.PUT("/attendance/:id/rating", a.UpdateAttendanceRating)
}
