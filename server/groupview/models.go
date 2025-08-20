package groupview

import "time"

// Group represents a family group
type Group struct {
	ID        string    `gorm:"primaryKey" json:"id"` // uuid string
	Name      string    `gorm:"not null" json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// GroupMember links users to groups with roles
type GroupMember struct {
	ID      string    `gorm:"primaryKey" json:"id"` // uuid string
	GroupID string    `gorm:"index;not null" json:"groupId"`
	UserID  uint      `gorm:"index;not null" json:"userId"`
	Role    string    `gorm:"not null;default:'member'" json:"role"` // 'admin' or 'member'
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Legacy Member model - keeping for backward compatibility during transition
type Member struct {
	ID          string    `gorm:"primaryKey" json:"id"` // uuid string
	Slug        string    `gorm:"uniqueIndex;not null" json:"slug"`
	DisplayName string    `gorm:"not null" json:"displayName"`
	IsActive    bool      `gorm:"not null;default:true" json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ViewingSession struct {
	ID        string     `gorm:"primaryKey" json:"id"` // uuid
	MediaID   string     `gorm:"index;not null" json:"mediaId"`
	MediaType string     `gorm:"not null" json:"mediaType"` // "movie" | "episode"
	StartedAt time.Time  `gorm:"not null" json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
	Source    *string    `json:"source"`
	Notes     *string    `json:"notes"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type Attendance struct {
	ID               string     `gorm:"primaryKey" json:"id"` // uuid
	ViewingSessionID string     `gorm:"index;not null" json:"viewingSessionId"`
	MemberID         string     `gorm:"index;not null" json:"memberId"` // Keep for backward compatibility
	UserID           *uint      `gorm:"index" json:"userId"`             // New field for user-based attendance
	Rating           *float64   `json:"rating"`
	RatedAt          *time.Time `json:"ratedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (Attendance) TableName() string    { return "attendances" }
func (ViewingSession) TableName() string { return "viewing_sessions" }
