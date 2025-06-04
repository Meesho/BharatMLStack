package auth

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type Repository interface {
	GetAllUsers() ([]User, error)
	GetUserByID(id uint) (*User, error)
	GetUserByEmailId(emailId string) (*User, error)
	CreateUser(user *User) (uint, error)
	UpdateUser(user *User) error
	DeleteUser(id uint) error
	UpdateUserAccessAndRole(email string, isActive bool, role string) error
}

type Auth struct {
	db     *gorm.DB
	dbName string
}

func NewRepository(connection *infra.SQLConnection) (Repository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}
	meta, err := connection.GetMeta()
	if err != nil {
		return nil, err
	}
	dbName := meta["db_name"].(string)

	return &Auth{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

// GetAllUsers retrieves all users from the database.
func (auth *Auth) GetAllUsers() ([]User, error) {
	var users []User
	result := auth.db.Find(&users)
	return users, result.Error
}

// GetUserByID retrieves a user by their ID.
func (auth *Auth) GetUserByID(id uint) (*User, error) {
	var user User
	result := auth.db.Where("id = ?", id).First(&user)
	return &user, result.Error
}

// GetUserByUsername retrieves a user by their username.
func (auth *Auth) GetUserByEmailId(emailId string) (*User, error) {
	var user User
	result := auth.db.Where("email = ?", emailId).First(&user)
	return &user, result.Error
}

// CreateUser adds a new user to the database.
func (auth *Auth) CreateUser(user *User) (uint, error) {
	result := auth.db.Create(user)
	if result.Error != nil {
		return 0, result.Error
	}
	return user.ID, nil
}

// UpdateUser updates a user's information in the database.
func (auth *Auth) UpdateUser(user *User) error {
	result := auth.db.Model(user).Where("id = ?", user.ID).Updates(user)
	return result.Error
}

// DeleteUser removes a user from the database by their ID.
func (auth *Auth) DeleteUser(id uint) error {
	result := auth.db.Where("id = ?", id).Delete(&User{})
	return result.Error
}

func (auth *Auth) UpdateUserAccessAndRole(email string, isActive bool, role string) error {
	var user User
	request := auth.db.Where("email = ?", email).First(&user)
	if request.Error != nil {
		return request.Error
	}
	var result *gorm.DB
	if user.Role != role {
		result = auth.db.Model(&User{}).Where("email = ?", email).Update("is_active", isActive).Update("role", role)
	} else {
		result = auth.db.Model(&User{}).Where("email = ?", email).Update("is_active", isActive)
	}
	return result.Error
}
