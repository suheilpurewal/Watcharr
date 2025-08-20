package groupview

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type API struct {
	DB *gorm.DB
}

// UserMember represents a user as a group member
type UserMember struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	IsActive    bool   `json:"isActive"`
}

func (a *API) GetMembers(c *gin.Context) {
	// Get all registered users and return them as group members
	var users []struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	}
	if err := a.DB.Table("users").Select("id, username").Find(&users).Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Convert users to member format
	var members []UserMember
	for _, user := range users {
		members = append(members, UserMember{
			ID:          fmt.Sprintf("%d", user.ID), // Convert uint to string
			DisplayName: user.Username,
			IsActive:    true, // All registered users are considered active
		})
	}

	c.JSON(http.StatusOK, members)
}

type postMemberIn struct {
	Slug        string `json:"slug" binding:"required"`
	DisplayName string `json:"displayName" binding:"required"`
}

func (a *API) PostMember(c *gin.Context) {
	var in postMemberIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.String(http.StatusBadRequest, err.Error()); return
	}
	m := Member{
		ID:          uuid.NewString(),
		Slug:        in.Slug,
		DisplayName: in.DisplayName,
		IsActive:    true,
	}
	if err := a.DB.Create(&m).Error; err != nil {
		c.String(http.StatusBadRequest, err.Error()); return
	}
	c.JSON(http.StatusCreated, gin.H{"id": m.ID})
}

type attendeeIn struct {
	MemberID string   `json:"memberId" binding:"required"`
	Rating   *float64 `json:"rating"`
}
type postViewingIn struct {
	MediaID   string       `json:"mediaId" binding:"required"`
	MediaType string       `json:"mediaType" binding:"required"` // "movie" | "episode"
	StartedAt time.Time    `json:"startedAt" binding:"required"` // ISO string okay
	Notes     *string      `json:"notes"`
	Attendees []attendeeIn `json:"attendees"`
}

func (a *API) PostViewing(c *gin.Context) {
	var in postViewingIn
	if err := c.ShouldBindJSON(&in); err != nil {
		c.String(http.StatusBadRequest, err.Error()); return
	}
	s := ViewingSession{
		ID:        uuid.NewString(),
		MediaID:   in.MediaID,
		MediaType: in.MediaType,
		StartedAt: in.StartedAt,
		Notes:     in.Notes,
	}
	tx := a.DB.Begin()
	if err := tx.Create(&s).Error; err != nil { tx.Rollback(); c.String(http.StatusBadRequest, err.Error()); return }
	for _, aIn := range in.Attendees {
		var ratedAt *time.Time
		if aIn.Rating != nil {
			t := time.Now().UTC()
			ratedAt = &t
		}
		
		// Convert member ID (which is now user ID string) to uint
		var userID *uint
		if aIn.MemberID != "" {
			var parsedID uint
			if n, err := fmt.Sscanf(aIn.MemberID, "%d", &parsedID); err == nil && n == 1 {
				userID = &parsedID // Successfully parsed user ID
			} else {
				// If parsing fails, treat as legacy member ID
				userID = nil
			}
		}
		
		rec := Attendance{
			ID:               uuid.NewString(),
			ViewingSessionID: s.ID,
			MemberID:         aIn.MemberID, // Keep for backward compatibility
			UserID:           userID,       // New user-based field
			Rating:           aIn.Rating,
			RatedAt:          ratedAt,
		}
		if err := tx.Create(&rec).Error; err != nil { tx.Rollback(); c.String(http.StatusBadRequest, err.Error()); return }
	}
	if err := tx.Commit().Error; err != nil { c.String(http.StatusInternalServerError, err.Error()); return }
	c.JSON(http.StatusCreated, gin.H{"id": s.ID})
}

type historyRow struct {
	SessionID  string    `json:"sessionID"`
	MediaID    string    `json:"mediaID"`
	MediaType  string    `json:"mediaType"`
	StartedAt  time.Time `json:"startedAt"`
	Notes      *string   `json:"notes"`
	MemberName string    `json:"memberName"`
	MemberID   string    `json:"memberID"`
	Rating     *float64  `json:"rating"`
}

func (a *API) GetHistory(c *gin.Context) {
	type historyRow struct {
		SessionID  string    `json:"sessionID"`
		MediaID    string    `json:"mediaID"`
		MediaType  string    `json:"mediaType"`
		StartedAt  time.Time `json:"startedAt"`
		Notes      *string   `json:"notes"`
		MemberName string    `json:"memberName"`
		MemberID   string    `json:"memberID"`
		Rating     *float64  `json:"rating"`
	}

	var rows []historyRow
	
	// Query that handles both user-based and legacy member-based attendance
	q := a.DB.Table("viewing_sessions AS vs").
		Select(`vs.id AS session_id, vs.media_id, vs.media_type, vs.started_at, vs.notes,
		        COALESCE(u.username, m.display_name) AS member_name, 
		        COALESCE(CAST(a.user_id AS TEXT), a.member_id) AS member_id, 
		        a.rating`).
		Joins("JOIN attendances a ON a.viewing_session_id = vs.id").
		Joins("LEFT JOIN users u ON u.id = a.user_id").
		Joins("LEFT JOIN members m ON m.id = a.member_id")

	// OPTIONAL FILTERS
	if mid := c.Query("mediaId"); mid != "" {
		q = q.Where("vs.media_id = ?", mid)
	}
	if mt := c.Query("mediaType"); mt != "" {
		q = q.Where("vs.media_type = ?", mt) // "movie" | "episode"
	}

	// default order / cap
	q = q.Order("vs.started_at DESC, COALESCE(u.username, m.display_name) ASC").Limit(500)

	if err := q.Scan(&rows).Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, rows)
}

// PendingRating represents an attendance record that needs a rating
type PendingRating struct {
	AttendanceID string    `json:"attendanceId"`
	SessionID    string    `json:"sessionId"`
	MediaID      string    `json:"mediaId"`
	MediaType    string    `json:"mediaType"`
	StartedAt    time.Time `json:"startedAt"`
	Notes        *string   `json:"notes"`
}

// GetMyPendingRatings returns unrated attendances for the current user
func (a *API) GetMyPendingRatings(c *gin.Context) {
	// Get current user ID from auth middleware
	userID, exists := c.Get("userId")
	if !exists {
		c.String(http.StatusUnauthorized, "User not authenticated")
		return
	}

	var pendingRatings []PendingRating
	
	// Find attendance records for this user that don't have ratings
	q := a.DB.Table("viewing_sessions AS vs").
		Select(`a.id AS attendance_id, vs.id AS session_id, vs.media_id, vs.media_type, 
		        vs.started_at, vs.notes`).
		Joins("JOIN attendances a ON a.viewing_session_id = vs.id").
		Where("a.user_id = ? AND a.rating IS NULL", userID).
		Order("vs.started_at DESC")

	if err := q.Scan(&pendingRatings).Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	
	c.JSON(http.StatusOK, pendingRatings)
}

// UpdateAttendanceRating allows a user to rate their attendance
func (a *API) UpdateAttendanceRating(c *gin.Context) {
	attendanceID := c.Param("id")
	userID, exists := c.Get("userId")
	if !exists {
		c.String(http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req struct {
		Rating float64 `json:"rating" binding:"required,min=0,max=10"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	// Update the attendance record if it belongs to the current user
	ratedAt := time.Now().UTC()
	result := a.DB.Model(&Attendance{}).
		Where("id = ? AND user_id = ?", attendanceID, userID).
		Updates(map[string]interface{}{
			"rating":   req.Rating,
			"rated_at": ratedAt,
		})

	if result.Error != nil {
		c.String(http.StatusInternalServerError, result.Error.Error())
		return
	}

	if result.RowsAffected == 0 {
		c.String(http.StatusNotFound, "Attendance record not found or not owned by user")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
