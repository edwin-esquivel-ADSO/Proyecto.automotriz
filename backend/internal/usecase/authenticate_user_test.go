package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"workshop/internal/domain"
	"workshop/internal/usecase"
)

type fakeUserRepository struct {
	user map[string]domain.User
}

func newFakeUserRepository(t *testing.T, username, password string) *fakeUserRepository {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hashing the fixture password failed: %v", err)
	}
	user, err := domain.NewUser("user-1", username, string(hash), "Administrador del taller", domain.RoleAdministrator, time.Now())
	if err != nil {
		t.Fatalf("building the user fixture failed: %v", err)
	}
	return &fakeUserRepository{user: map[string]domain.User{username: user}}
}

func (f *fakeUserRepository) FindByUsername(_ context.Context, username string) (domain.User, error) {
	found, ok := f.user[username]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return found, nil
}

func (f *fakeUserRepository) FindByID(_ context.Context, id string) (domain.User, error) {
	for _, item := range f.user {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (f *fakeUserRepository) UpdatePassword(_ context.Context, userID, passwordHash string) error {
	for k, item := range f.user {
		if item.ID == userID {
			item.PasswordHash = passwordHash
			item.RequiresPasswordChange = false
			f.user[k] = item
			return nil
		}
	}
	return domain.ErrNotFound
}

type fakeTokenIssuer struct {
	issued int
}

func (f *fakeTokenIssuer) Issue(userID string, _ domain.Role, issuedAt time.Time) (string, time.Time, error) {
	f.issued++
	return "token-for-" + userID, issuedAt.Add(time.Hour), nil
}

func TestAuthenticateIssuesASessionForValidCredentials(t *testing.T) {
	users := newFakeUserRepository(t, "admin", "Admin2026")
	tokens := &fakeTokenIssuer{}
	useCase := usecase.NewAuthenticateUser(users, tokens, fixedClock())

	session, err := useCase.Execute(context.Background(), "admin", "Admin2026")
	if err != nil {
		t.Fatalf("valid credentials must be accepted: %v", err)
	}
	if session.Token != "token-for-user-1" {
		t.Fatalf("the session must carry the issued token, got %q", session.Token)
	}
	if session.Role != domain.RoleAdministrator {
		t.Fatalf("the session must carry the role of the user, got %s", session.Role)
	}
}

func TestAuthenticateRejectsAWrongPasswordWithoutIssuingAToken(t *testing.T) {
	users := newFakeUserRepository(t, "admin", "Admin2026")
	tokens := &fakeTokenIssuer{}
	useCase := usecase.NewAuthenticateUser(users, tokens, fixedClock())

	_, err := useCase.Execute(context.Background(), "admin", "wrong-password")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("a wrong password must be rejected as unauthorized, got %v", err)
	}
	if tokens.issued != 0 {
		t.Fatalf("no token may be issued for a failed sign in, got %d", tokens.issued)
	}
}

func TestAuthenticateRejectsAnUnknownUserWithTheSameError(t *testing.T) {
	users := newFakeUserRepository(t, "admin", "Admin2026")
	useCase := usecase.NewAuthenticateUser(users, &fakeTokenIssuer{}, fixedClock())

	_, unknownErr := useCase.Execute(context.Background(), "ghost", "Admin2026")
	_, wrongPasswordErr := useCase.Execute(context.Background(), "admin", "nope")
	if !errors.Is(unknownErr, domain.ErrUnauthorized) || !errors.Is(wrongPasswordErr, domain.ErrUnauthorized) {
		t.Fatalf("both failures must be unauthorized, got %v and %v", unknownErr, wrongPasswordErr)
	}
	if unknownErr.Error() != wrongPasswordErr.Error() {
		t.Fatal("an unknown user and a wrong password must fail identically, so the response reveals nothing")
	}
}

func TestAuthenticateRejectsEmptyCredentials(t *testing.T) {
	useCase := usecase.NewAuthenticateUser(newFakeUserRepository(t, "admin", "Admin2026"), &fakeTokenIssuer{}, fixedClock())
	if _, err := useCase.Execute(context.Background(), "", ""); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("empty credentials must be rejected as unauthorized, got %v", err)
	}
}

func TestBootstrapCredentialsHashes(t *testing.T) {
	passwords := map[string]string{
		"admin":    "Admin2026*",
		"jperez":   "JPerez2026*",
		"lramirez": "LRamirez2026*",
	}
	for user, pass := range passwords {
		h, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			t.Fatalf("failed to hash password for %s: %v", user, err)
		}
		if err := bcrypt.CompareHashAndPassword(h, []byte(pass)); err != nil {
			t.Fatalf("hash verification failed for %s: %v", user, err)
		}
	}
}

func TestExactUserCredentialsAuthentication(t *testing.T) {
	usersMap := make(map[string]domain.User)
	passwords := map[string]struct {
		pass string
		role domain.Role
	}{
		"admin":    {pass: "Admin2026*", role: domain.RoleAdministrator},
		"jperez":   {pass: "JPerez2026*", role: domain.RoleTechnician},
		"lramirez": {pass: "LRamirez2026*", role: domain.RoleTechnician},
	}
	for u, data := range passwords {
		hash, err := bcrypt.GenerateFromPassword([]byte(data.pass), bcrypt.MinCost)
		if err != nil {
			t.Fatalf("hashing failed for %s: %v", u, err)
		}
		user, err := domain.NewUser("id-"+u, u, string(hash), u+" full name", data.role, time.Now())
		if err != nil {
			t.Fatalf("creating user failed for %s: %v", u, err)
		}
		usersMap[u] = user
	}
	repo := &fakeUserRepository{user: usersMap}
	useCase := usecase.NewAuthenticateUser(repo, &fakeTokenIssuer{}, fixedClock())

	for u, data := range passwords {
		session, err := useCase.Execute(context.Background(), u, data.pass)
		if err != nil {
			t.Fatalf("expected successful login for %s: %v", u, err)
		}
		if session.Username != u {
			t.Errorf("expected username %s, got %s", u, session.Username)
		}
		if session.Role != data.role {
			t.Errorf("expected role %s for %s, got %s", data.role, u, session.Role)
		}
	}
}

func TestChangePassword(t *testing.T) {
	users := newFakeUserRepository(t, "admin", "Admin2026*")
	useCase := usecase.NewAuthenticateUser(users, &fakeTokenIssuer{}, fixedClock())

	// Test wrong current password
	err := useCase.ChangePassword(context.Background(), "user-1", "WrongOld*", "NewSecret2026*")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for wrong current password, got %v", err)
	}

	// Test weak new password
	err = useCase.ChangePassword(context.Background(), "user-1", "Admin2026*", "weak")
	if !errors.Is(err, domain.ErrWeakPassword) {
		t.Fatalf("expected ErrWeakPassword for short password, got %v", err)
	}

	// Test successful change
	err = useCase.ChangePassword(context.Background(), "user-1", "Admin2026*", "NewSecret2026*")
	if err != nil {
		t.Fatalf("expected successful password change, got %v", err)
	}

	// Verify new password works
	_, err = useCase.Execute(context.Background(), "admin", "NewSecret2026*")
	if err != nil {
		t.Fatalf("login with new password failed: %v", err)
	}
}


