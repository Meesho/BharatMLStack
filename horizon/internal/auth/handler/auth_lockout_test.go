package handler

import (
	"errors"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/auth"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// fakeAuthRepo is a minimal auth.Repository for exercising Login. It returns
// the configured user for a matching email, gorm.ErrRecordNotFound for any
// other email, and a configurable transient error when lookupErr is set.
type fakeAuthRepo struct {
	user      *auth.User
	lookupErr error
}

func (f *fakeAuthRepo) GetAllUsers() ([]auth.User, error) { return nil, nil }
func (f *fakeAuthRepo) GetUserByID(id uint) (*auth.User, error) {
	return nil, nil
}
func (f *fakeAuthRepo) GetUserByEmailId(emailId string) (*auth.User, error) {
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	if f.user != nil && f.user.Email == emailId {
		return f.user, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeAuthRepo) CreateUser(user *auth.User) (uint, error)                     { return 1, nil }
func (f *fakeAuthRepo) UpdateUser(user *auth.User) error                             { return nil }
func (f *fakeAuthRepo) DeleteUser(id uint) error                                     { return nil }
func (f *fakeAuthRepo) UpdateUserAccessAndRole(email string, a bool, r string) error { return nil }

// fakeTokenRepo is a no-op token.Repository.
type fakeTokenRepo struct{}

func (fakeTokenRepo) SaveToken(email, token string, expiration time.Time) error { return nil }
func (fakeTokenRepo) InvalidateToken(token string) error                        { return nil }
func (fakeTokenRepo) IsTokenValid(token string) (bool, error)                   { return true, nil }
func (fakeTokenRepo) CleanupExpiredTokens() error                               { return nil }

func newTestHandler(u *auth.User) *AuthHandler {
	return &AuthHandler{authRepo: &fakeAuthRepo{user: u}, tokenRepo: fakeTokenRepo{}}
}

func TestLogin_LocksAfterRepeatedFailures(t *testing.T) {
	email := "lockme@example.com"
	resetLoginFailures(email) // isolate from other tests

	hash, _ := bcrypt.GenerateFromPassword([]byte("CorrectHorse1!"), bcrypt.DefaultCost)
	h := newTestHandler(&auth.User{Email: email, PasswordHash: string(hash), IsActive: true})

	// Hit the wrong password maxLoginFailures times.
	for i := 0; i < maxLoginFailures; i++ {
		if _, err := h.Login(&Login{Email: email, Password: "wrong"}); err == nil {
			t.Fatalf("attempt %d: expected error for wrong password", i+1)
		}
	}

	// Now the account should be locked even with the *correct* password.
	if _, err := h.Login(&Login{Email: email, Password: "CorrectHorse1!"}); err == nil {
		t.Fatal("expected account to be locked after repeated failures")
	}

	resetLoginFailures(email) // cleanup
}

func TestLogin_SuccessResetsFailureCounter(t *testing.T) {
	email := "resetme@example.com"
	resetLoginFailures(email)

	hash, _ := bcrypt.GenerateFromPassword([]byte("CorrectHorse1!"), bcrypt.DefaultCost)
	h := newTestHandler(&auth.User{Email: email, PasswordHash: string(hash), IsActive: true})

	// A few failures, but below the threshold.
	for i := 0; i < maxLoginFailures-1; i++ {
		_, _ = h.Login(&Login{Email: email, Password: "wrong"})
	}
	// A successful login should clear the counter.
	if _, err := h.Login(&Login{Email: email, Password: "CorrectHorse1!"}); err != nil {
		t.Fatalf("expected successful login, got %v", err)
	}
	if locked, _ := isAccountLocked(email); locked {
		t.Fatal("account should not be locked after a successful login")
	}

	resetLoginFailures(email)
}

func TestIsAccountLocked_Unknown(t *testing.T) {
	if locked, _ := isAccountLocked("never-seen@example.com"); locked {
		t.Fatal("unknown account should not be locked")
	}
}

// TestLogin_UnknownEmailLocksLikeKnown verifies that an email with no backing
// account still locks after the threshold and reports the *same* generic
// "locked" error a known account would. This keeps the lockout from becoming an
// account-existence oracle.
func TestLogin_UnknownEmailLocksLikeKnown(t *testing.T) {
	email := "ghost@example.com"
	resetLoginFailures(email)

	h := newTestHandler(nil) // no user exists; lookups return ErrRecordNotFound

	for i := 0; i < maxLoginFailures; i++ {
		if _, err := h.Login(&Login{Email: email, Password: "whatever"}); err == nil {
			t.Fatalf("attempt %d: expected error for unknown email", i+1)
		}
	}

	locked, _ := isAccountLocked(email)
	if !locked {
		t.Fatal("unknown email should be locked after crossing the threshold")
	}

	resetLoginFailures(email)
}

// TestLogin_TransientErrorDoesNotLock verifies that a transient backend error
// (anything other than ErrRecordNotFound) is not counted as a failed login, so
// an infra blip cannot lock out a legitimate user.
func TestLogin_TransientErrorDoesNotLock(t *testing.T) {
	email := "flaky-db@example.com"
	resetLoginFailures(email)

	h := &AuthHandler{
		authRepo:  &fakeAuthRepo{lookupErr: errors.New("connection refused")},
		tokenRepo: fakeTokenRepo{},
	}

	for i := 0; i < maxLoginFailures+2; i++ {
		if _, err := h.Login(&Login{Email: email, Password: "whatever"}); err == nil {
			t.Fatalf("attempt %d: expected transient error", i+1)
		}
	}

	if locked, _ := isAccountLocked(email); locked {
		t.Fatal("account must not be locked due to transient backend errors")
	}

	resetLoginFailures(email)
}
