package groupview

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"log/slog"
)

type API struct {
	DB *gorm.DB
}

// WatchedStatus represents the status of watched content
type WatchedStatus string

const (
	FINISHED WatchedStatus = "FINISHED"
	WATCHING WatchedStatus = "WATCHING"
	PLANNED  WatchedStatus = "PLANNED"
	HOLD     WatchedStatus = "HOLD"
	DROPPED  WatchedStatus = "DROPPED"
)

// Watched represents a watched item in the database
type Watched struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt *time.Time     `json:"deletedAt" gorm:"index"`
	Status    WatchedStatus  `json:"status"`
	Rating    float64        `json:"rating" gorm:"type:numeric(2,1)"`
	UserID    uint           `json:"-" gorm:"uniqueIndex:usernctnidx"`
	ContentID *int           `json:"-" gorm:"uniqueIndex:usernctnidx"`
}

// UserMember represents a user as a group member
type UserMember struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	IsActive    bool   `json:"isActive"`
}

// FamilyHistoryItem represents a family viewing session with attendees
type FamilyHistoryItem struct {
	SessionID    string    `json:"sessionId"`
	MediaID      string    `json:"mediaId"`
	MediaType    string    `json:"mediaType"`
	StartedAt    time.Time `json:"startedAt"`
	Notes        *string   `json:"notes"`
	AttendeeCount int      `json:"attendeeCount"`
	AverageRating *float64 `json:"averageRating"`
			Attendees    []struct {
			UserID   uint    `json:"userId"`
			Username string  `json:"username"`
			Rating   float64 `json:"rating"`
		} `json:"attendees"`
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
		
		// Create a Watched record for this user/content combination (without rating)
		// The rating will be added later through the regular Watcharr rating system
		if userID != nil {
			// Convert mediaID to integer for the watcheds table
			mediaIDInt, err := strconv.Atoi(in.MediaID)
			if err == nil {
				// Check if a watched record already exists
				var existingWatched Watched
				watchedExists := tx.Where("user_id = ? AND content_id = ?", *userID, mediaIDInt).First(&existingWatched).Error == nil
				
				if !watchedExists {
					// Create new watched record without rating (rating will be added later)
					watchedRecord := Watched{
						Status:    FINISHED,
						Rating:    0.0, // Default to 0, will be updated when user rates
						UserID:    *userID,
						ContentID: &mediaIDInt,
					}
					
					if err := tx.Create(&watchedRecord).Error; err != nil {
						tx.Rollback()
						c.String(http.StatusBadRequest, "Failed to create watched record: "+err.Error())
						return
					}
				}
				// If record exists, don't update it - let the regular rating system handle it
			}
		}
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

// GetGroupInfo returns information about the user's family group
func (a *API) GetGroupInfo(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.String(http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Find the user's group through group_members table
	var groupInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Role string `json:"role"`
	}

	q := a.DB.Table("groups AS g").
		Select("g.id, g.name, gm.role").
		Joins("JOIN group_members gm ON gm.group_id = g.id").
		Where("gm.user_id = ?", userID)

	if err := q.Scan(&groupInfo).Error; err != nil {
		c.String(http.StatusNotFound, "No group found for user")
		return
	}

	c.JSON(http.StatusOK, groupInfo)
}

// UpdateGroupName allows group admins to update the group name
func (a *API) UpdateGroupName(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.String(http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req struct {
		Name string `json:"name" binding:"required,min=1,max=50"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	// Check if user is admin of their group
	var groupMember GroupMember
	if err := a.DB.Where("user_id = ? AND role = ?", userID, "admin").Take(&groupMember).Error; err != nil {
		c.String(http.StatusForbidden, "Only group admins can update group name")
		return
	}

	// Update the group name
	result := a.DB.Model(&Group{}).
		Where("id = ?", groupMember.GroupID).
		Update("name", req.Name)

	if result.Error != nil {
		c.String(http.StatusInternalServerError, result.Error.Error())
		return
	}

	if result.RowsAffected == 0 {
		c.String(http.StatusNotFound, "Group not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "name": req.Name})
}

// GetFamilyHistory returns all family viewing sessions for the user's group
func (a *API) GetFamilyHistory(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.String(http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Get user's group ID
	var groupMember GroupMember
	if err := a.DB.Where("user_id = ?", userID).Take(&groupMember).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// User is not in any group, return empty history
			slog.Info("GetFamilyHistory: User not in any group, returning empty history", "userID", userID)
			c.JSON(http.StatusOK, []FamilyHistoryItem{})
			return
		}
		slog.Error("GetFamilyHistory: Database error checking group membership", "userID", userID, "error", err)
		c.String(http.StatusInternalServerError, "Failed to check group membership")
		return
	}

	slog.Info("GetFamilyHistory: Found user in group", "userID", userID, "groupID", groupMember.GroupID)

	var history []FamilyHistoryItem

	// Get all viewing sessions with attendees from the user's group
	slog.Info("GetFamilyHistory: Starting query for group", "groupID", groupMember.GroupID)
	
	// First, get the basic session information
	type sessionRow struct {
		SessionID      string     `json:"session_id"`
		MediaID        string     `json:"media_id"`
		MediaType      string     `json:"media_type"`
		StartedAt      time.Time  `json:"started_at"`
		Notes          *string    `json:"notes"`
		AttendeeCount  int        `json:"attendee_count"`
		AverageRating  *float64   `json:"average_rating"`
	}
	
	var sessionRows []sessionRow
	q := a.DB.Table("viewing_sessions AS vs").
		Select(`vs.id AS session_id, vs.media_id, vs.media_type, vs.started_at, vs.notes,
		        COUNT(a.id) AS attendee_count, AVG(w.rating) AS average_rating`).
		Joins("JOIN attendances a ON a.viewing_session_id = vs.id").
		Joins("JOIN group_members gm ON gm.user_id = a.user_id").
		Joins("LEFT JOIN watcheds w ON w.user_id = a.user_id AND CAST(w.content_id AS TEXT) = vs.media_id AND w.status = 'FINISHED'").
		Where("gm.group_id = ? AND a.user_id IS NOT NULL", groupMember.GroupID).
		Group("vs.id, vs.media_id, vs.media_type, vs.started_at, vs.notes").
		Order("vs.started_at DESC").
		Limit(100)

	if err := q.Scan(&sessionRows).Error; err != nil {
		slog.Error("GetFamilyHistory query failed", "error", err, "userID", userID, "groupID", groupMember.GroupID)
		c.String(http.StatusInternalServerError, "Failed to load family history")
		return
	}
	
	slog.Info("GetFamilyHistory: Found sessions", "sessionCount", len(sessionRows), "groupID", groupMember.GroupID)
	
	// Debug: Check if watcheds table exists and has data
	var watchedCount int64
	a.DB.Model(&struct{}{}).Table("watcheds").Count(&watchedCount)
	slog.Info("Debug: watcheds table count", "count", watchedCount)
	
			// Debug: Show all records in watcheds table
		var watchedRecords []struct {
			UserID    uint    `json:"user_id"`
			ContentID int     `json:"content_id"`
			Status    string  `json:"status"`
			Rating    float64 `json:"rating"`
		}
		a.DB.Table("watcheds").Select("user_id, content_id, status, rating").Find(&watchedRecords)
		slog.Info("Debug: All watcheds records", "records", watchedRecords)
		
		// Debug: Show actual rating values for the specific content
		if len(sessionRows) > 0 {
			mediaIDInt, _ := strconv.Atoi(sessionRows[0].MediaID)
			var specificRatings []struct {
				UserID uint    `json:"user_id"`
				Rating float64 `json:"rating"`
			}
			a.DB.Table("watcheds").Select("user_id, rating").Where("content_id = ? AND status = 'FINISHED'", mediaIDInt).Find(&specificRatings)
			slog.Info("Debug: Specific ratings for content", "contentID", mediaIDInt, "ratings", specificRatings)
			
			// Debug: Show all watcheds records for each user to see what content_ids they have
			var userWatchedRecords []struct {
				UserID    uint    `json:"user_id"`
				ContentID int     `json:"content_id"`
				Rating    float64 `json:"rating"`
			}
			a.DB.Table("watcheds").Select("user_id, content_id, rating").Where("status = 'FINISHED'").Find(&userWatchedRecords)
			slog.Info("Debug: All user watched records by user", "records", userWatchedRecords)
		}
	
			// Debug: Check for specific ratings
		var testRating struct {
			Rating float64
		}
		// Convert mediaID to integer for the query
		mediaIDInt, _ := strconv.Atoi(sessionRows[0].MediaID)
		testQuery := a.DB.Table("watcheds").Select("rating").Where("user_id = ? AND content_id = ? AND status = 'FINISHED'", userID, mediaIDInt).First(&testRating)
		if testQuery.Error != nil {
			slog.Warn("Debug: No rating found for test query", "error", testQuery.Error, "userID", userID, "contentID", mediaIDInt)
		} else {
			slog.Info("Debug: Found test rating", "rating", testRating.Rating)
		}

	// Convert session rows to FamilyHistoryItem
	history = make([]FamilyHistoryItem, len(sessionRows))
	for i, row := range sessionRows {
		history[i] = FamilyHistoryItem{
			SessionID:      row.SessionID,
			MediaID:        row.MediaID,
			MediaType:      row.MediaType,
			StartedAt:      row.StartedAt,
			Notes:          row.Notes,
			AttendeeCount:  row.AttendeeCount,
			AverageRating:  row.AverageRating,
			Attendees:      []struct {
				UserID   uint    `json:"userId"`
				Username string  `json:"username"`
				Rating   float64 `json:"rating"`
			}{},
		}
	}

	// Get attendees for each session
	for i := range history {
		var attendees []struct {
			UserID   uint    `json:"userId"`
			Username string  `json:"username"`
			Rating   float64 `json:"rating"`
		}

		// Get ALL attendees for this session with their ratings from the Watched table
		// First, let's find what content_id is actually stored in watcheds for this user/content combination
		attendeeQuery := a.DB.Table("attendances AS a").
			Select("u.id AS user_id, u.username, COALESCE(w.rating, 0) AS rating").
			Joins("JOIN users u ON u.id = a.user_id").
			Joins("LEFT JOIN watcheds w ON w.user_id = u.id AND w.status = 'FINISHED'").
			Where("a.viewing_session_id = ? AND a.user_id IS NOT NULL", 
				history[i].SessionID)
		
		// Debug: Log the SQL query
		sql := attendeeQuery.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Find(&[]struct{}{})
		})
		slog.Info("Attendee query SQL", "sql", sql, "mediaID", history[i].MediaID, "sessionID", history[i].SessionID)

		if err := attendeeQuery.Scan(&attendees).Error; err != nil {
			slog.Warn("Failed to get attendees for session", "sessionID", history[i].SessionID, "error", err)
			continue
		}

		slog.Info("Found attendees for session", "sessionID", history[i].SessionID, "attendeeCount", len(attendees), "attendees", attendees)
		history[i].Attendees = attendees
	}

	c.JSON(http.StatusOK, history)
}

// GetPersonalHistory returns all viewing sessions where the current user was in attendance
func (a *API) GetPersonalHistory(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.String(http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Get all viewing sessions where the current user was in attendance
	type sessionRow struct {
		SessionID      string     `json:"session_id"`
		MediaID        string     `json:"media_id"`
		MediaType      string     `json:"media_type"`
		StartedAt      time.Time  `json:"started_at"`
		Notes          *string    `json:"notes"`
		AttendeeCount  int        `json:"attendee_count"`
		AverageRating  *float64   `json:"average_rating"`
		UserRating     *float64   `json:"user_rating"`
	}
	
	var sessionRows []sessionRow
	q := a.DB.Table("viewing_sessions AS vs").
		Select(`vs.id AS session_id, vs.media_id, vs.media_type, vs.started_at, vs.notes,
		        COUNT(a.id) AS attendee_count, AVG(a.rating) AS average_rating,
		        user_attendance.rating AS user_rating`).
		Joins("JOIN attendances a ON a.viewing_session_id = vs.id").
		Joins("JOIN attendances user_attendance ON user_attendance.viewing_session_id = vs.id").
		Where("user_attendance.user_id = ? AND a.user_id IS NOT NULL", userID).
		Group("vs.id, vs.media_id, vs.media_type, vs.started_at, vs.notes, user_attendance.rating").
		Order("vs.started_at DESC").
		Limit(100)

	if err := q.Scan(&sessionRows).Error; err != nil {
		slog.Error("GetPersonalHistory query failed", "error", err, "userID", userID)
		c.String(http.StatusInternalServerError, "Failed to load personal history")
		return
	}

	// Convert to FamilyHistoryItem format for consistency
	history := make([]FamilyHistoryItem, len(sessionRows))
	for i, row := range sessionRows {
		history[i] = FamilyHistoryItem{
			SessionID:      row.SessionID,
			MediaID:        row.MediaID,
			MediaType:      row.MediaType,
			StartedAt:      row.StartedAt,
			Notes:          row.Notes,
			AttendeeCount:  row.AttendeeCount,
			AverageRating:  row.AverageRating,
			Attendees:      []struct {
				UserID   uint    `json:"userId"`
				Username string  `json:"username"`
				Rating   float64 `json:"rating"`
			}{},
		}
	}

	// Get attendees for each session
	for i := range history {
		var attendees []struct {
			UserID   uint    `json:"userId"`
			Username string  `json:"username"`
			Rating   float64 `json:"rating"`
		}

		attendeeQuery := a.DB.Table("attendances AS a").
			Select("u.id AS user_id, u.username, a.rating").
			Joins("JOIN users u ON u.id = a.user_id").
			Where("a.viewing_session_id = ? AND a.user_id IS NOT NULL", history[i].SessionID)

		if err := attendeeQuery.Scan(&attendees).Error; err != nil {
			slog.Warn("Failed to get attendees for session", "sessionID", history[i].SessionID, "error", err)
			continue
		}

		history[i].Attendees = attendees
	}

	c.JSON(http.StatusOK, history)
}

// ShareContentToFamily shares a user's personal watched content to the family group
func (a *API) ShareContentToFamily(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.String(http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req struct {
		MediaID   string    `json:"mediaId" binding:"required"`
		MediaType string    `json:"mediaType" binding:"required"`
		StartedAt time.Time `json:"startedAt" binding:"required"`
		Notes     *string   `json:"notes"`
		Rating    *float64  `json:"rating"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	// Get user's group
	var groupMember GroupMember
	if err := a.DB.Where("user_id = ?", userID).Take(&groupMember).Error; err != nil {
		c.String(http.StatusNotFound, "User not in any group")
		return
	}

	// Create viewing session
	session := ViewingSession{
		ID:        uuid.NewString(),
		MediaID:   req.MediaID,
		MediaType: req.MediaType,
		StartedAt: req.StartedAt,
		Notes:     req.Notes,
	}

	tx := a.DB.Begin()
	if err := tx.Create(&session).Error; err != nil {
		tx.Rollback()
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Create attendance record for the sharing user
	var ratedAt *time.Time
	if req.Rating != nil {
		t := time.Now().UTC()
		ratedAt = &t
	}

	// Convert userID to uint properly
	userIDUint := userID.(uint)
	attendance := Attendance{
		ID:               uuid.NewString(),
		ViewingSessionID: session.ID,
		UserID:           &userIDUint,
		Rating:           req.Rating,
		RatedAt:          ratedAt,
	}

	if err := tx.Create(&attendance).Error; err != nil {
		tx.Rollback()
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"sessionId": session.ID})
}

// EnsureUserInGroup ensures the current user is in a family group, creating one if needed
func (a *API) EnsureUserInGroup(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.String(http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Check if user is already in a group
	var groupMember GroupMember
	if err := a.DB.Where("user_id = ?", userID).Take(&groupMember).Error; err == nil {
		// User is already in a group
		c.JSON(http.StatusOK, gin.H{"message": "User already in group", "groupId": groupMember.GroupID})
		return
	}

	// User is not in a group, check if any group exists
	var existingGroup Group
	if err := a.DB.First(&existingGroup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No group exists, create the first family group and make this user admin
			groupID := uuid.New().String()
			newGroup := Group{
				ID:   groupID,
				Name: "Family",
			}
			
			if err := a.DB.Create(&newGroup).Error; err != nil {
				slog.Error("EnsureUserInGroup: Failed to create family group", "error", err)
				c.String(http.StatusInternalServerError, "Failed to create family group")
				return
			}
			
			// Create group member record for this user as admin
			memberID := uuid.New().String()
			groupMember := GroupMember{
				ID:      memberID,
				GroupID: groupID,
				UserID:  userID.(uint),
				Role:    "admin",
			}
			
			if err := a.DB.Create(&groupMember).Error; err != nil {
				slog.Error("EnsureUserInGroup: Failed to create group member record", "error", err)
				c.String(http.StatusInternalServerError, "Failed to create group member record")
				return
			}
			
			c.JSON(http.StatusOK, gin.H{"message": "Created new family group and made user admin", "groupId": groupID})
			return
		} else {
			slog.Error("EnsureUserInGroup: Database error checking for existing groups", "error", err)
			c.String(http.StatusInternalServerError, "Database error")
			return
		}
	}

	// Group exists, add this user as a member
	memberID := uuid.New().String()
	groupMember = GroupMember{
		ID:      memberID,
		GroupID: existingGroup.ID,
		UserID:  userID.(uint),
		Role:    "member",
	}
	
	if err := a.DB.Create(&groupMember).Error; err != nil {
		slog.Error("EnsureUserInGroup: Failed to create group member record", "error", err)
		c.String(http.StatusInternalServerError, "Failed to create group member record")
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Added user to existing family group", "groupId": existingGroup.ID})
}

// DebugGroupStatus returns debug information about the user's group status
func (a *API) DebugGroupStatus(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.String(http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Check if tables exist
	var tableCount int
	a.DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('groups', 'group_members', 'viewing_sessions', 'attendances')").Scan(&tableCount)
	
	// Check if user is in a group
	var groupMember GroupMember
	var inGroup bool
	var groupID string
	if err := a.DB.Where("user_id = ?", userID).Take(&groupMember).Error; err == nil {
		inGroup = true
		groupID = groupMember.GroupID
	}
	
	// Count records in each table
	var groupCount, memberCount, sessionCount, attendanceCount int64
	a.DB.Model(&Group{}).Count(&groupCount)
	a.DB.Model(&GroupMember{}).Count(&memberCount)
	a.DB.Model(&ViewingSession{}).Count(&sessionCount)
	a.DB.Model(&Attendance{}).Count(&attendanceCount)
	
	c.JSON(http.StatusOK, gin.H{
		"userId": userID,
		"inGroup": inGroup,
		"groupId": groupID,
		"tablesExist": tableCount == 4,
		"tableCounts": gin.H{
			"groups": groupCount,
			"groupMembers": memberCount,
			"viewingSessions": sessionCount,
			"attendances": attendanceCount,
		},
	})
}

// TestDatabaseTables is a simple endpoint to check if the database tables exist (no auth required)
func (a *API) TestDatabaseTables(c *gin.Context) {
	// Check if tables exist
	var tableCount int
	a.DB.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('groups', 'group_members', 'viewing_sessions', 'attendances')").Scan(&tableCount)
	
	// Count records in each table
	var groupCount, memberCount, sessionCount, attendanceCount int64
	a.DB.Model(&Group{}).Count(&groupCount)
	a.DB.Model(&GroupMember{}).Count(&memberCount)
	a.DB.Model(&ViewingSession{}).Count(&sessionCount)
	a.DB.Model(&Attendance{}).Count(&attendanceCount)
	
	c.JSON(http.StatusOK, gin.H{
		"tablesExist": tableCount == 4,
		"tableCounts": gin.H{
			"groups": groupCount,
			"groupMembers": memberCount,
			"viewingSessions": sessionCount,
			"attendances": attendanceCount,
		},
	})
}
