package groupview

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type API struct {
	DB *gorm.DB
}

func (a *API) GetMembers(c *gin.Context) {
	var ms []Member
	if err := a.DB.Order("display_name asc").Find(&ms).Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error()); return
	}
	c.JSON(http.StatusOK, ms)
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
		rec := Attendance{
			ID:               uuid.NewString(),
			ViewingSessionID: s.ID,
			MemberID:         aIn.MemberID,
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
	q := a.DB.Table("viewing_sessions AS vs").
		Select(`vs.id AS session_id, vs.media_id, vs.media_type, vs.started_at, vs.notes,
		        m.display_name AS member_name, a.member_id, a.rating`).
		Joins("JOIN attendances a ON a.viewing_session_id = vs.id").
		Joins("JOIN members m ON m.id = a.member_id")

	// OPTIONAL FILTERS
	if mid := c.Query("mediaId"); mid != "" {
		q = q.Where("vs.media_id = ?", mid)
	}
	if mt := c.Query("mediaType"); mt != "" {
		q = q.Where("vs.media_type = ?", mt) // "movie" | "episode"
	}

	// default order / cap
	q = q.Order("vs.started_at DESC, m.display_name ASC").Limit(500)

	if err := q.Scan(&rows).Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, rows)
}
