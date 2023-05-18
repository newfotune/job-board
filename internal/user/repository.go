package user

import (
	"database/sql"
	"errors"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/segmentio/ksuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db}
}

func (r *Repository) SaveTokenSignOn(email, token, userType string) error {
	if _, err := r.db.Exec(`INSERT INTO user_sign_on_token (token, email, user_type, created_at) VALUES ($1, $2, $3, NOW())`, token, email, userType); err != nil {
		return err
	}
	return nil
}

func (r *Repository) GetUser(user_id string) (*User, error) {
	row := r.db.QueryRow(`SELECT id, email, created_at, user_type, email_verified, access_token, refresh_token, expiration_time FROM users where id = $1`, user_id)
	var id, email, userType, accessToken, refreshToken sql.NullString
	var createdAt, expirationTime sql.NullTime
	var emailVerified sql.NullBool
	err := row.Scan(&id, &email, &createdAt, &userType, &emailVerified, &accessToken, &refreshToken, &expirationTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &User{
		ID:            id.String,
		Email:         email.String,
		EmailVerified: emailVerified.Bool,
		AccessToken:   accessToken.String,
		RefreshToken:  refreshToken.String,
		CreatedAt:     createdAt.Time,
		Type:          userType.String,
	}, nil
}

func (r *Repository) CreateUser(u User) error {
	_, err := r.db.Exec(
		`INSERT INTO users (id, email, created_at, user_type, email_verified, access_token, refresh_token, expiration_time) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		u.ID, u.Email, u.CreatedAt, u.Type, u.EmailVerified, u.AccessToken, u.RefreshToken, u.ExpirationTime)
	return err
}

func (r *Repository) UpdateAccessToken(userId, accessToken string) error {
	_, err := r.db.Exec(`UPDATE users SET access_token = $1 WHERE id = $2`, accessToken, userId)
	return err
}

func (r *Repository) UpdateRefreshToken(userId, refreshToken string) error {
	_, err := r.db.Exec(`UPDATE users SET refresh_token = $1 WHERE id = $2`, refreshToken, userId)
	return err
}

// GetOrCreateUserFromToken creates or get existing user given a token
// returns the user struct, whether the user existed already and an error
func (r *Repository) GetOrCreateUserFromToken(token string) (User, bool, error) {
	u := User{}
	row := r.db.QueryRow(`SELECT id, email, created_at, user_type, email_verified, access_token, expiration_time FROM users where token = $1`, token)
	//row := r.db.QueryRow(`SELECT t.token, t.email, u.id, u.email, u.created_at, t.user_type
	// FROM user_sign_on_token t LEFT JOIN users u ON t.email = u.email WHERE t.token = $1`, token)
	var id, email, userType, accessToken sql.NullString
	var createdAt, expirationTime sql.NullTime
	var emailVerified sql.NullBool
	if err := row.Scan(&id, &email, &createdAt, &userType, &emailVerified, &accessToken, &expirationTime); err != nil {
		return u, false, err
	}
	if !accessToken.Valid {
		return u, false, errors.New("token not found")
	}
	if !email.Valid {
		// user not found create new one
		userID, err := ksuid.NewRandom()
		if err != nil {
			return u, false, err
		}
		u.ID = userID.String()
		u.Email = accessToken.String
		u.CreatedAt = time.Now()
		u.Type = userType.String
		u.CreatedAtHumanised = humanize.Time(u.CreatedAt.UTC())
		if _, err := r.db.Exec(`INSERT INTO users (id, email, created_at, user_type) VALUES ($1, $2, $3, $4)`, u.ID, u.Email, u.CreatedAt, u.Type); err != nil {
			return User{}, false, err
		}

		return u, false, nil
	}
	u.ID = id.String
	u.Email = email.String
	u.CreatedAt = createdAt.Time
	u.Type = userType.String
	u.CreatedAtHumanised = humanize.Time(u.CreatedAt.UTC())

	return u, true, nil
}

func (r *Repository) DeleteUserByEmail(email string) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE email = $1`, email)
	return err
}

// DeleteExpiredUserSignOnTokens deletes user_sign_on_tokens older than 1 week
func (r *Repository) DeleteExpiredUserSignOnTokens() error {
	_, err := r.db.Exec(`DELETE FROM user_sign_on_token WHERE created_at < NOW() - INTERVAL '7 DAYS'`)
	return err
}

func (r *Repository) GetUserTypeByEmail(email string) (string, error) {
	var userType string
	row := r.db.QueryRow(`SELECT user_type FROM users WHERE email = $1`, email)
	err := row.Scan(&userType)
	if err == sql.ErrNoRows {
		// check if user is unverified recruiter/developer
		row = r.db.QueryRow(`SELECT 'recruiter' FROM recruiter_profile WHERE email = $1`, email)
		err = row.Scan(&userType)
		if err == nil {
			return userType, nil
		}
		row = r.db.QueryRow(`SELECT 'developer' FROM developer_profile WHERE email = $1`, email)
		err = row.Scan(&userType)
		if err == nil {
			return userType, nil
		}
		return userType, err
	}
	if err != nil {
		return userType, err
	}
	return userType, nil
}
