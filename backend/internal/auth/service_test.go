package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository é um mock para a interface UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(ctx context.Context, name, email, passwordHash string) (*User, error) {
	args := m.Called(ctx, name, email, passwordHash)
	if args.Get(0) != nil {
		return args.Get(0).(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) != nil {
		return args.Get(0).(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) GetUserByIDWithHash(ctx context.Context, id string) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, id, name, email string) (*User, error) {
	args := m.Called(ctx, id, name, email)
	if args.Get(0) != nil {
		return args.Get(0).(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	return m.Called(ctx, id, passwordHash).Error(0)
}

func (m *MockUserRepository) DeleteUser(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func setupService() (*Service, *MockUserRepository, redismock.ClientMock) {
	repoMock := new(MockUserRepository)
	db, mockRedis := redismock.NewClientMock()
	service := NewService(repoMock, db, "secret")
	return service, repoMock, mockRedis
}

func TestService_Register(t *testing.T) {
	t.Run("Empty fields", func(t *testing.T) {
		s, _, _ := setupService()
		_, err := s.Register(context.Background(), "", "", "pass")
		assert.EqualError(t, err, "todos os campos são obrigatórios")
	})

	t.Run("Short password", func(t *testing.T) {
		s, _, _ := setupService()
		_, err := s.Register(context.Background(), "test", "test@test.com", "123")
		assert.EqualError(t, err, "a senha deve ter no mínimo 6 caracteres")
	})

	t.Run("Email already registered", func(t *testing.T) {
		s, repo, _ := setupService()
		repo.On("GetUserByEmail", mock.Anything, "test@test.com").Return(&User{}, nil)
		_, err := s.Register(context.Background(), "Test", "test@test.com", "password")
		assert.EqualError(t, err, "este e-mail já está cadastrado")
	})

	t.Run("Success", func(t *testing.T) {
		s, repo, _ := setupService()
		repo.On("GetUserByEmail", mock.Anything, "test@test.com").Return(nil, errors.New("not found"))
		repo.On("CreateUser", mock.Anything, "Test", "test@test.com", mock.AnythingOfType("string")).Return(&User{ID: "1"}, nil)

		user, err := s.Register(context.Background(), "Test", "test@test.com", "password")
		assert.NoError(t, err)
		assert.Equal(t, "1", user.ID)
		repo.AssertExpectations(t)
	})
}

func TestService_Login(t *testing.T) {
	t.Run("User not found", func(t *testing.T) {
		s, repo, _ := setupService()
		repo.On("GetUserByEmail", mock.Anything, "test@test.com").Return(nil, errors.New("not found"))
		_, _, _, err := s.Login(context.Background(), "test@test.com", "password")
		assert.EqualError(t, err, "e-mail ou senha incorretos")
	})

	t.Run("Wrong password", func(t *testing.T) {
		s, repo, _ := setupService()
		hash, _ := hashPassword("right_password", defaultParams)
		repo.On("GetUserByEmail", mock.Anything, "test@test.com").Return(&User{PasswordHash: hash}, nil)
		
		_, _, _, err := s.Login(context.Background(), "test@test.com", "wrong_password")
		assert.EqualError(t, err, "e-mail ou senha incorretos")
	})

	t.Run("Success", func(t *testing.T) {
		s, repo, rdbMock := setupService()
		hash, _ := hashPassword("password", defaultParams)
		user := &User{ID: "1", Email: "test@test.com", PasswordHash: hash}
		repo.On("GetUserByEmail", mock.Anything, "test@test.com").Return(user, nil)
		
		rdbMock.Regexp().ExpectSet("^refresh_token:.*", "1", 7*24*time.Hour).SetVal("OK")

		resUser, access, refresh, err := s.Login(context.Background(), "test@test.com", "password")
		assert.NoError(t, err)
		if resUser != nil {
			assert.Equal(t, "1", resUser.ID)
		}
		assert.NotEmpty(t, access)
		assert.NotEmpty(t, refresh)
	})
}

func TestService_GenerateRefreshToken(t *testing.T) {
	s, _, rdbMock := setupService()
	rdbMock.Regexp().ExpectSet("^refresh_token:.*", "1", 7*24*time.Hour).SetErr(errors.New("redis error"))
	
	_, err := s.GenerateRefreshToken(context.Background(), "1")
	assert.EqualError(t, err, "redis error")
}

func TestService_ValidateRefreshToken(t *testing.T) {
	s, _, rdbMock := setupService()
	rdbMock.ExpectGet("refresh_token:valid").SetVal("1")
	
	id, err := s.ValidateRefreshToken(context.Background(), "valid")
	assert.NoError(t, err)
	assert.Equal(t, "1", id)
}

func TestService_RevokeRefreshToken(t *testing.T) {
	s, _, rdbMock := setupService()
	rdbMock.ExpectDel("refresh_token:token").SetVal(1)
	
	err := s.RevokeRefreshToken(context.Background(), "token")
	assert.NoError(t, err)
}

func TestService_GetUserByID(t *testing.T) {
	s, repo, _ := setupService()
	repo.On("GetUserByID", mock.Anything, "1").Return(&User{ID: "1"}, nil)
	
	user, err := s.GetUserByID(context.Background(), "1")
	assert.NoError(t, err)
	assert.Equal(t, "1", user.ID)
}

func TestComparePasswordAndHash(t *testing.T) {
	hash, err := hashPassword("test1234", defaultParams)
	assert.NoError(t, err)

	match, err := comparePasswordAndHash("test1234", hash)
	assert.NoError(t, err)
	assert.True(t, match)

	match, err = comparePasswordAndHash("wrong", hash)
	assert.NoError(t, err)
	assert.False(t, match)

	// Invalid format
	_, err = comparePasswordAndHash("test1234", "invalid")
	assert.Error(t, err)
}

func TestService_UpdateProfile(t *testing.T) {
	s, repo, _ := setupService()
	repo.On("GetUserByEmail", mock.Anything, "new@test.com").Return(nil, errors.New("not found"))
	repo.On("UpdateUser", mock.Anything, "1", "NewName", "new@test.com").Return(&User{ID: "1", Name: "NewName", Email: "new@test.com"}, nil)

	user, err := s.UpdateProfile(context.Background(), "1", "NewName", "new@test.com")
	assert.NoError(t, err)
	assert.Equal(t, "NewName", user.Name)
	assert.Equal(t, "new@test.com", user.Email)
}

func TestService_UpdatePassword(t *testing.T) {
	s, repo, _ := setupService()
	hash, _ := hashPassword("oldpassword", defaultParams)
	repo.On("GetUserByIDWithHash", mock.Anything, "1").Return(&User{ID: "1", PasswordHash: hash}, nil)
	repo.On("UpdatePassword", mock.Anything, "1", mock.Anything).Return(nil)

	err := s.UpdatePassword(context.Background(), "1", "oldpassword", "newpassword")
	assert.NoError(t, err)
}

func TestService_DeleteUser(t *testing.T) {
	s, repo, _ := setupService()
	repo.On("DeleteUser", mock.Anything, "1").Return(nil)

	err := s.DeleteUser(context.Background(), "1")
	assert.NoError(t, err)
}
