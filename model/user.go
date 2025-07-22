package model

import (
	"time"

	"github.com/chatmcp/mcprouter/util"
	"gorm.io/gorm"
)

type User struct {
	UUID           string    `json:"uuid"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	DeletedAt      time.Time `json:"-"`
	Email          string    `json:"email"`
	Nickname       string    `json:"nickname"`
	AvatarURL      string    `json:"avatar_url"`
	SigninType     string    `json:"signin_type"`
	SigninProvider string    `json:"signin_provider"`
	SigninOpenid   string    `json:"signin_openid"`
	SigninIP       string    `json:"signin_ip"`
}

func (u *User) TableName() string {
	return "users"
}

func CreateUser(user *User) error {
	return db().Create(user).Error
}

// SaveUser saves the user information using email as the unique key.
// If a user with the given email exists, it updates the record; otherwise, it creates a new one.
// The operation is performed within a transaction.
func SaveUser(user *User) error {
	return db().Transaction(func(tx *gorm.DB) error {
		var existing User
		// Query user by email
		err := tx.Where("email = ?", user.Email).First(&existing).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// User does not exist, create new
				user.UUID = util.GenUUID()
				if err := tx.Create(user).Error; err != nil {
					return err
				}
			} else {
				// Other error
				return err
			}
		} else {
			// User exists, update
			user.UUID = existing.UUID
			user.CreatedAt = existing.CreatedAt
			user.UpdatedAt = time.Now()

			if err := tx.Model(&existing).Where("email = ?", user.Email).Updates(user).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func FindUserByEmail(email string) (*User, error) {
	user := &User{}
	if err := db().Where("email = ?", email).First(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func FindUserByUUID(uuid string) (*User, error) {
	user := &User{}
	if err := db().Where("uuid = ?", uuid).First(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}
