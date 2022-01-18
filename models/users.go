package models

import (
	"errors"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/uhdang/lenslocked/hash"
	"github.com/uhdang/lenslocked/rand"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrNotFound is returned when a resource cannot be found // in the database.
	ErrNotFound = errors.New("models: resource not found")

	// ErrInvalidID is returned when an invalid ID is provided to a method like Delete.
	ErrInvalidID = errors.New("models: ID provided was invalid")

	// ErrInvalidPassword is returned when an invalid password
	//is used when attempting to authenticate a user.
	ErrInvalidPassword = errors.New("models: incorrect password provided")

	userPwPepper = "random-string-as-pepper"
)

var _ UserDB = &userGorm{}
var _ UserService = &userService{}

const hmacSecretKey = "secret-hmac-key"

type User struct {
	gorm.Model
	Name         string
	Email        string `gorm:"not null;unique_index"`
	Password     string `gorm:"-"`
	PasswordHash string `gorm:"not null"`
	Remember     string `gorm:"-"`
	RememberHash string `gorm:"not null;unique_index"`
	Age          int
}

// UserService is a set of methods used to manipulate and
// work with the user model
type UserService interface {
	// Authenticate will verify the provided email address and
	// password are correct. If they are correct, the user
	// corresponding to that email will be returned. Otherwise
	// You will receive either:
	// ErrNotFound, ErrInvalidPassword, or another error if
	// something goes wrong
	Authenticate(email, password string) (*User, error)
	UserDB
}

type userService struct {
	UserDB
}

// UserDB is used to interact with the user database.
type UserDB interface {
	// Methods for quering for single users
	ByID(id uint) (*User, error)
	ByEmail(email string) (*User, error)
	ByRemember(token string) (*User, error)

	// Methods for altering users
	Create(user *User) error
	Update(user *User) error
	Delete(id uint) error

	// Used to close a DB connection
	Close() error

	// Migration helpers
	AutoMigrate() error
	DestructiveReset() error
}

//func (uv *userValidator) ByID(id uint) (*User, error) {
//	// Validate the ID
//	if id <= 0 {
//		return nil, errors.New("Invalid ID")
//	}
//	// if it is valid, call the next method in the chain and
//	// return its results.
//	return uv.UserDB.ByID(id)
//}

// Authenticate can be used to authenticate a user with the
//provided email address and password.
// If the email address provided is invalid, this will return
//		nil, ErrNotFound
// If the password provided is invalid, this will return
//		nil, ErrInvalidPassword
// If the email and password are both valid, this will return
//		user, nil
// Otherwise if another error is encountered this will return
//		nil, error
func (us *userService) Authenticate(email, password string) (*User, error) {
	foundUser, err := us.ByEmail(email)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(foundUser.PasswordHash),
		[]byte(password+userPwPepper))
	switch err {
	case nil:
		return foundUser, nil
	case bcrypt.ErrMismatchedHashAndPassword:
		return nil, ErrInvalidPassword
	default:
		return nil, err
	}
}

func NewUserService(connectionInfo string) (UserService, error) {
	ug, err := newUserGorm(connectionInfo)
	if err != nil {
		return nil, err
	}
	hmac := hash.NewHMAC(hmacSecretKey)
	uv := &userValidator{
		hmac:   hmac,
		UserDB: ug,
	}
	return &userService{
		UserDB: uv,
	}, nil
}

// userValidator is our validation layer that validates
// and normalizes data before passing it on to the next
// UserDB in our interface chain.
type userValidator struct {
	UserDB
	hmac hash.HMAC
}

// userValFn defines a single format for validation functions
type userValFn func(*User) error

func runUserValFns(user *User, fns ...userValFn) error {
	for _, fn := range fns {
		if err := fn(user); err != nil {
			return err
		}
	}
	return nil
}

// bcryptPassword will hash a user's password with an
// app-wide pepper and bcrypt, which salts for us.
func (uv *userValidator) bcryptPassword(user *User) error {
	if user.Password == "" {
		// We DO NOT need to run this if the password
		// hasn't been changed
		return nil
	}
	pwBytes := []byte(user.Password + userPwPepper)
	hashedBytes, err := bcrypt.GenerateFromPassword(pwBytes, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hashedBytes)
	user.Password = ""
	return nil
}

// hmacRemember we check to see if the remember token was set before proceeding.
// If it was not set, we terminate early returning nil. Otherwise we hash the
// remember token that is present
// like all other validation functions, this will accept a pointer to a user
// as its only argument and return an error if the validation fails.
func (uv *userValidator) hmacRemember(user *User) error {
	if user.Remember == "" {
		return nil
	}
	user.RememberHash = uv.hmac.Hash(user.Remember)
	return nil
}

func (uv *userValidator) setRememberIfUnset(user *User) error {
	if user.Remember != "" {
		return nil
	}
	token, err := rand.RememberToken()
	if err != nil {
		return err
	}
	user.Remember = token
	return nil
}

func (uv *userValidator) idGreaterThan(n uint) userValFn {
	return userValFn(func(user *User) error {
		if user.ID <= n {
			return ErrInvalidID
		}
		return nil
	})
}

// ByRemember will hash the remember token and then call
// ByRemember on the subsequent UserDB layer.
func (uv *userValidator) ByRemember(token string) (*User, error) {
	user := User{
		Remember: token,
	}
	if err := runUserValFns(&user, uv.hmacRemember); err != nil {
		return nil, err
	}
	return uv.UserDB.ByRemember(user.RememberHash)
}

// Create will create the provided user and backfill data
// lke the ID, CreateAt, and UpdatedAt fields.
func (uv *userValidator) Create(user *User) error {
	err := runUserValFns(user,
		uv.bcryptPassword,
		uv.setRememberIfUnset,
		uv.hmacRemember,
	)
	if err != nil {
		return err
	}
	return uv.UserDB.Create(user)
}

// Update will hash a remember token if it is provided
func (uv *userValidator) Update(user *User) error {
	err := runUserValFns(user,
		uv.bcryptPassword,
		uv.hmacRemember)
	if err != nil {
		return err
	}
	return uv.UserDB.Update(user)
}

// Delete will delete the user with the provided ID
func (uv *userValidator) Delete(id uint) error {
	var user User
	user.ID = id
	err := runUserValFns(&user, uv.idGreaterThan(0))
	if err != nil {
		return err
	}
	return uv.UserDB.Delete(id)
}

// userGorm represents our database interaction layer
// and implements the UserDB interface fully.
type userGorm struct {
	db *gorm.DB
}

func newUserGorm(connectionInfo string) (*userGorm, error) {
	db, err := gorm.Open("postgres", connectionInfo)
	if err != nil {
		return nil, err
	}
	db.LogMode(true)
	return &userGorm{
		db: db,
	}, nil
}

// Closes the UserService database connection
func (ug *userGorm) Close() error {
	return ug.db.Close()
}

// ByID will look up a user with the provided ID.
// If the user is found, we will return a nil error
// If the user is not found, we will return ErrNotFound
// If there is another error, we will return an error with more information about what went wrong.
// This may not be an error generated by the model package.
//
// As a general rule, any error but ErrNotFound should probably result in a 500 error.
func (ug *userGorm) ByID(id uint) (*User, error) {
	var user User
	db := ug.db.Where("id = ?", id)
	err := first(db, &user)
	if err != nil {
		return nil, err
	}
	return &user, err
}

// ByEmail will look up a user with the provided email
func (ug *userGorm) ByEmail(email string) (*User, error) {
	var user User
	db := ug.db.Where("email = ?", email)
	err := first(db, &user)
	return &user, err
}

// ByAge will look up the first user with given age
func (ug *userGorm) ByAge(age int) (*User, error) {
	var user User
	db := ug.db.Where("age = ?", age)
	err := first(db, &user)
	if err != nil {
		return nil, err
	}
	return &user, err
}

// ByRemember looks up a user with the given remember token
// and returns that user. This method will handle hashing
// the token for us.
// Errors are the same as ByEmail.
func (ug *userGorm) ByRemember(rememberHash string) (*User, error) {
	var user User
	err := first(ug.db.Where("remember_hash = ?", rememberHash), &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// DestructiveReset drops the user table and rebuilds it
func (ug *userGorm) DestructiveReset() error {
	err := ug.db.DropTableIfExists(&User{}).Error
	if err != nil {
		return err
	}
	return ug.AutoMigrate()
}

// Create will create the provided user and backfill data like the ID, CreatedAt, and UpdatedAt fields.
func (ug *userGorm) Create(user *User) error {
	return ug.db.Create(user).Error
}

// first will query the provided gorm.DB and it will get the first item returned
// and place it into dst. If nothing is found in the query, it will return ErrNotFound
func first(db *gorm.DB, dst interface{}) error {
	err := db.First(dst).Error
	if err == gorm.ErrRecordNotFound {
		return ErrNotFound
	}
	return err
}

// Update will update the provided user with all data in the provided user object.
func (ug *userGorm) Update(user *User) error {
	return ug.db.Save(user).Error
}

// Delete will delete the user with the provided ID
func (ug *userGorm) Delete(id uint) error {
	user := User{Model: gorm.Model{ID: id}}
	return ug.db.Delete(&user).Error
}

// AutoMigrate will attempt to automatically migrate the users table
func (ug *userGorm) AutoMigrate() error {
	if err := ug.db.AutoMigrate(&User{}).Error; err != nil {
		return err
	}
	return nil
}
