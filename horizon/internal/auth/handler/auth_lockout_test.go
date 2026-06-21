package handler

import (
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/auth"
	"golang.org/x/crypto/bcrypt"
)

// fakeAuthRepo is a minimal auth.Repository for exercising Login.
type fakeAuthRepo struct {
	user *auth.User
}

func (f *fakeAuthRepo) GetAllUsers() ([]auth.User, error) { return nil, nil }
func (f *fakeAuthRepo) GetUserByID(id uint) (*auth.User, error) {
	return nil, nil
}
func (f *fakeAuthRepo) GetUserByEmailId(emailId string) (*auth.User, error) {
	if f.user != nil && f.user.Email == emailId {
		return f.user, nil
	}
	return nil, errGorm
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

var errGorm = errorString("record not found")

type errorString string

func (e errorString) Error() string { return string(e) }

func newTestHandler(u *auth.User) *AuthHandler {
	return &AuthHandler{authRepo: &fakeAuthRepo{user: u}, tokenRepo: fakeTokenRepo{}}
}

func TestLogin_LocksAfterRepeatedFailures(t *testing.T) {
	email := "lockme@example.com"
	loginFailures.Delete(email) // isolate from other tests

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

	loginFailures.Delete(email) // cleanup
}

func TestLogin_SuccessResetsFailureCounter(t *testing.T) {
	email := "resetme@example.com"
	loginFailures.Delete(email)

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

	loginFailures.Delete(email)
}

func TestIsAccountLocked_Unknown(t *testing.T) {
	if locked, _ := isAccountLocked("never-seen@example.com"); locked {
		t.Fatal("unknown account should not be locked")
	}
}
