package user

import "time"

const (
	UserTypeDeveloper = "jobseeker"    // TODO: Change to employee
	UserTypeAdmin     = "admin"        // TODO: Unused remove
	UserTypeRecruiter = "workerseeker" // TODO: Change to employer
)

type User struct {
	ID                 string
	Email              string
	EmailVerified      bool
	AccessToken        string
	RefreshToken       string
	ExpirationTime     time.Time
	CreatedAt          time.Time
	Type               string
	IsAdmin            bool // Not sure how this is used.
	CreatedAtHumanised string
}
