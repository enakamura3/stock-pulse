package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/golang-jwt/jwt/v5"
	"github.com/onigiri/stock-pulse/backend/internal/config"
	"github.com/redis/go-redis/v9"
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

		rdbMock.Regexp().ExpectSet("^refresh_token:.*", "1", 12*time.Hour).SetVal("OK")
		rdbMock.Regexp().ExpectSAdd("^user_active_tokens:1$", ".*").SetVal(1)
		rdbMock.ExpectExpire("user_active_tokens:1", 12*time.Hour).SetVal(true)

		resUser, access, refresh, err := s.Login(context.Background(), "test@test.com", "password")
		assert.NoError(t, err)
		if resUser != nil {
			assert.Equal(t, "1", resUser.ID)
		}
		assert.NotEmpty(t, access)
		assert.NotEmpty(t, refresh)
	})
}

func TestService_Login_RefreshTokenError(t *testing.T) {
	s, repo, rdbMock := setupService()
	hash, _ := hashPassword("password", defaultParams)
	user := &User{ID: "1", Email: "test@test.com", PasswordHash: hash}
	repo.On("GetUserByEmail", mock.Anything, "test@test.com").Return(user, nil)
	rdbMock.Regexp().ExpectSet("^refresh_token:.*", "1", 12*time.Hour).SetErr(errors.New("redis err"))

	_, _, _, err := s.Login(context.Background(), "test@test.com", "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "falha ao gerar refresh token")
}

func TestService_Login_AccessTokenError(t *testing.T) {
	repo := new(MockUserRepository)
	db, _ := redismock.NewClientMock()
	s := NewService(repo, db, "") // Secret vazio faz GenerateAccessToken falhar
	hash, _ := hashPassword("password", defaultParams)
	user := &User{ID: "1", Email: "test@test.com", PasswordHash: hash}
	repo.On("GetUserByEmail", mock.Anything, "test@test.com").Return(user, nil)

	_, _, _, err := s.Login(context.Background(), "test@test.com", "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "falha ao gerar access token")
}

func TestService_GenerateRefreshToken_Errors(t *testing.T) {
	t.Run("SAdd error", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.Regexp().ExpectSet("^refresh_token:.*", "user-1", 12*time.Hour).SetVal("OK")
		rdbMock.Regexp().ExpectSAdd("^user_active_tokens:user-1$", ".*").SetErr(errors.New("sadd err"))

		_, err := s.GenerateRefreshToken(context.Background(), "user-1")
		assert.EqualError(t, err, "sadd err")
	})

	t.Run("Expire error", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.Regexp().ExpectSet("^refresh_token:.*", "user-1", 12*time.Hour).SetVal("OK")
		rdbMock.Regexp().ExpectSAdd("^user_active_tokens:user-1$", ".*").SetVal(1)
		rdbMock.ExpectExpire("user_active_tokens:user-1", 12*time.Hour).SetErr(errors.New("expire err"))

		_, err := s.GenerateRefreshToken(context.Background(), "user-1")
		assert.EqualError(t, err, "expire err")
	})
}

func TestService_ValidateRefreshToken(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("refresh_token:valid").SetVal("user-123")

		userID, err := s.ValidateRefreshToken(context.Background(), "valid")
		assert.NoError(t, err)
		assert.Equal(t, "user-123", userID)
	})

	t.Run("Expired", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("refresh_token:expired").SetErr(redis.Nil)

		_, err := s.ValidateRefreshToken(context.Background(), "expired")
		assert.EqualError(t, err, "sessão expirada ou inválida")
	})

	t.Run("Error", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("refresh_token:invalid").SetErr(errors.New("redis err"))

		_, err := s.ValidateRefreshToken(context.Background(), "invalid")
		assert.Error(t, err)
	})
}

func TestService_RotateRefreshToken(t *testing.T) {
	t.Run("Empty token", func(t *testing.T) {
		s, _, _ := setupService()
		_, _, err := s.RotateRefreshToken(context.Background(), "   ")
		assert.EqualError(t, err, "token inválido")
	})

	t.Run("Replay attack detected - multiple active tokens revoked", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("used_refresh_token:replayed_token").SetVal("user-123")
		rdbMock.ExpectSMembers("user_active_tokens:user-123").SetVal([]string{"active1", "active2"})
		rdbMock.ExpectDel("refresh_token:active1").SetVal(1)
		rdbMock.ExpectDel("refresh_token:active2").SetVal(1)
		rdbMock.ExpectDel("user_active_tokens:user-123").SetVal(1)

		userID, newRef, err := s.RotateRefreshToken(context.Background(), "replayed_token")
		assert.Empty(t, userID)
		assert.Empty(t, newRef)
		assert.EqualError(t, err, "tentativa de reutilização de token detectada")
	})

	t.Run("Replay attack detected - SMembers error still revokes user_active_tokens", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("used_refresh_token:replayed_token").SetVal("user-123")
		rdbMock.ExpectSMembers("user_active_tokens:user-123").SetErr(errors.New("smembers err"))
		rdbMock.ExpectDel("user_active_tokens:user-123").SetVal(1)

		userID, newRef, err := s.RotateRefreshToken(context.Background(), "replayed_token")
		assert.Empty(t, userID)
		assert.Empty(t, newRef)
		assert.EqualError(t, err, "tentativa de reutilização de token detectada")
	})

	t.Run("Redis error on checking used token", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("used_refresh_token:err_token").SetErr(errors.New("redis err"))

		_, _, err := s.RotateRefreshToken(context.Background(), "err_token")
		assert.EqualError(t, err, "redis err")
	})

	t.Run("Expired or non-existent token", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("used_refresh_token:non_existent").SetErr(redis.Nil)
		rdbMock.ExpectGet("refresh_token:non_existent").SetErr(redis.Nil)

		_, _, err := s.RotateRefreshToken(context.Background(), "non_existent")
		assert.EqualError(t, err, "sessão expirada ou inválida")
	})

	t.Run("Redis error on checking refresh token", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("used_refresh_token:token").SetErr(redis.Nil)
		rdbMock.ExpectGet("refresh_token:token").SetErr(errors.New("redis get err"))

		_, _, err := s.RotateRefreshToken(context.Background(), "token")
		assert.EqualError(t, err, "redis get err")
	})

	t.Run("Error setting used_refresh_token", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("used_refresh_token:valid_token").SetErr(redis.Nil)
		rdbMock.ExpectGet("refresh_token:valid_token").SetVal("user-123")
		rdbMock.ExpectDel("refresh_token:valid_token").SetVal(1)
		rdbMock.ExpectSRem("user_active_tokens:user-123", "valid_token").SetVal(1)
		rdbMock.ExpectSet("used_refresh_token:valid_token", "user-123", 12*time.Hour).SetErr(errors.New("redis set err"))

		_, _, err := s.RotateRefreshToken(context.Background(), "valid_token")
		assert.EqualError(t, err, "redis set err")
	})

	t.Run("Error generating new refresh token", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("used_refresh_token:valid_token").SetErr(redis.Nil)
		rdbMock.ExpectGet("refresh_token:valid_token").SetVal("user-123")
		rdbMock.ExpectDel("refresh_token:valid_token").SetVal(1)
		rdbMock.ExpectSRem("user_active_tokens:user-123", "valid_token").SetVal(1)
		rdbMock.ExpectSet("used_refresh_token:valid_token", "user-123", 12*time.Hour).SetVal("OK")
		rdbMock.Regexp().ExpectSet("^refresh_token:.*", "user-123", 12*time.Hour).SetErr(errors.New("generate err"))

		_, _, err := s.RotateRefreshToken(context.Background(), "valid_token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "falha ao gerar novo refresh token")
	})

	t.Run("Success rotation", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("used_refresh_token:old_token").SetErr(redis.Nil)
		rdbMock.ExpectGet("refresh_token:old_token").SetVal("user-123")
		rdbMock.ExpectDel("refresh_token:old_token").SetVal(1)
		rdbMock.ExpectSRem("user_active_tokens:user-123", "old_token").SetVal(1)
		rdbMock.ExpectSet("used_refresh_token:old_token", "user-123", 12*time.Hour).SetVal("OK")
		rdbMock.Regexp().ExpectSet("^refresh_token:.*", "user-123", 12*time.Hour).SetVal("OK")
		rdbMock.Regexp().ExpectSAdd("^user_active_tokens:user-123$", ".*").SetVal(1)
		rdbMock.ExpectExpire("user_active_tokens:user-123", 12*time.Hour).SetVal(true)

		userID, newRefreshToken, err := s.RotateRefreshToken(context.Background(), "old_token")
		assert.NoError(t, err)
		assert.Equal(t, "user-123", userID)
		assert.NotEmpty(t, newRefreshToken)
		assert.NotEqual(t, "old_token", newRefreshToken)
	})
}

func TestService_RevokeRefreshToken(t *testing.T) {
	t.Run("Success with user in token", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("refresh_token:token").SetVal("user-1")
		rdbMock.ExpectSRem("user_active_tokens:user-1", "token").SetVal(1)
		rdbMock.ExpectDel("used_refresh_token:token").SetVal(1)
		rdbMock.ExpectDel("refresh_token:token").SetVal(1)

		err := s.RevokeRefreshToken(context.Background(), "token")
		assert.NoError(t, err)
	})

	t.Run("Token not found in redis", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("refresh_token:token").SetErr(redis.Nil)
		rdbMock.ExpectDel("used_refresh_token:token").SetVal(0)
		rdbMock.ExpectDel("refresh_token:token").SetVal(0)

		err := s.RevokeRefreshToken(context.Background(), "token")
		assert.NoError(t, err)
	})
}

func TestService_GetUserByID(t *testing.T) {
	s, repo, _ := setupService()
	repo.On("GetUserByID", mock.Anything, "1").Return(&User{ID: "1"}, nil)

	user, err := s.GetUserByID(context.Background(), "1")
	assert.NoError(t, err)
	assert.Equal(t, "1", user.ID)
}

func TestService_GenerateAccessToken(t *testing.T) {
	s, _, _ := setupService()
	user := &User{ID: "user-123", Email: "user@test.com"}

	t.Run("Nil user", func(t *testing.T) {
		_, err := s.GenerateAccessToken(nil)
		assert.EqualError(t, err, "usuário não fornecido")
	})

	t.Run("Empty secret", func(t *testing.T) {
		sNoSecret := NewService(nil, nil, "")
		_, err := sNoSecret.GenerateAccessToken(user)
		assert.EqualError(t, err, "jwtSecret não configurado")
	})

	t.Run("Success and Expiration", func(t *testing.T) {
		tokenStr, err := s.GenerateAccessToken(user)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenStr)

		// Valida os claims do token gerado
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return s.jwtSecret, nil
		})
		assert.NoError(t, err)
		assert.True(t, token.Valid)

		claims, ok := token.Claims.(jwt.MapClaims)
		assert.True(t, ok)
		assert.Equal(t, "user-123", claims["user_id"])
		assert.Equal(t, "user@test.com", claims["email"])

		expFloat, ok := claims["exp"].(float64)
		assert.True(t, ok)
		expectedExp := time.Now().Add(15 * time.Minute).Unix()
		// Tolera diferença de até 5 segundos devido ao tempo de execução do teste
		assert.InDelta(t, expectedExp, int64(expFloat), 5)
	})
}

func TestService_NewService_CustomTTL(t *testing.T) {
	origAccess := config.Envs.JWTAccessTokenTTL
	origRefresh := config.Envs.JWTRefreshTokenTTL
	defer func() {
		config.Envs.JWTAccessTokenTTL = origAccess
		config.Envs.JWTRefreshTokenTTL = origRefresh
	}()

	config.Envs.JWTAccessTokenTTL = 30 * time.Minute
	config.Envs.JWTRefreshTokenTTL = 24 * time.Hour

	repoMock := new(MockUserRepository)
	db, _ := redismock.NewClientMock()
	service := NewService(repoMock, db, "secret")

	assert.Equal(t, 30*time.Minute, service.accessTokenTTL)
	assert.Equal(t, 24*time.Hour, service.refreshTokenTTL)
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

	// Invalid format (less than 6 parts)
	_, err = comparePasswordAndHash("test1234", "invalid")
	assert.Error(t, err)

	// Invalid params part
	_, err = comparePasswordAndHash("test1234", "$argon2id$v=19$badparams$c2FsdA$aGFzaA")
	assert.Error(t, err)

	// Invalid base64 salt
	_, err = comparePasswordAndHash("test1234", "$argon2id$v=19$m=65536,t=1,p=4$bad#salt$aGFzaA")
	assert.Error(t, err)

	// Invalid base64 hash
	_, err = comparePasswordAndHash("test1234", "$argon2id$v=19$m=65536,t=1,p=4$c2FsdA$bad#hash")
	assert.Error(t, err)
}

func TestService_UpdateProfile(t *testing.T) {
	t.Run("Empty fields", func(t *testing.T) {
		s, _, _ := setupService()
		_, err := s.UpdateProfile(context.Background(), "1", "", "test@test.com")
		assert.EqualError(t, err, "todos os campos são obrigatórios")
	})

	t.Run("Email already in use", func(t *testing.T) {
		s, repo, _ := setupService()
		repo.On("GetUserByEmail", mock.Anything, "other@test.com").Return(&User{ID: "2", Email: "other@test.com"}, nil)
		_, err := s.UpdateProfile(context.Background(), "1", "NewName", "other@test.com")
		assert.EqualError(t, err, "este e-mail já está cadastrado por outro usuário")
	})

	t.Run("Success", func(t *testing.T) {
		s, repo, _ := setupService()
		repo.On("GetUserByEmail", mock.Anything, "new@test.com").Return(nil, errors.New("not found"))
		repo.On("UpdateUser", mock.Anything, "1", "NewName", "new@test.com").Return(&User{ID: "1", Name: "NewName", Email: "new@test.com"}, nil)

		user, err := s.UpdateProfile(context.Background(), "1", "NewName", "new@test.com")
		assert.NoError(t, err)
		assert.Equal(t, "NewName", user.Name)
		assert.Equal(t, "new@test.com", user.Email)
	})
}

func TestService_UpdatePassword(t *testing.T) {
	t.Run("Short new password", func(t *testing.T) {
		s, _, _ := setupService()
		err := s.UpdatePassword(context.Background(), "1", "old", "123")
		assert.EqualError(t, err, "a nova senha deve ter no mínimo 6 caracteres")
	})

	t.Run("User not found", func(t *testing.T) {
		s, repo, _ := setupService()
		repo.On("GetUserByIDWithHash", mock.Anything, "1").Return(nil, errors.New("not found"))
		err := s.UpdatePassword(context.Background(), "1", "oldpassword", "newpassword")
		assert.Error(t, err)
	})

	t.Run("Wrong current password", func(t *testing.T) {
		s, repo, _ := setupService()
		hash, _ := hashPassword("rightpassword", defaultParams)
		repo.On("GetUserByIDWithHash", mock.Anything, "1").Return(&User{ID: "1", PasswordHash: hash}, nil)
		err := s.UpdatePassword(context.Background(), "1", "wrongpassword", "newpassword")
		assert.EqualError(t, err, "senha atual incorreta")
	})

	t.Run("Success", func(t *testing.T) {
		s, repo, _ := setupService()
		hash, _ := hashPassword("oldpassword", defaultParams)
		repo.On("GetUserByIDWithHash", mock.Anything, "1").Return(&User{ID: "1", PasswordHash: hash}, nil)
		repo.On("UpdatePassword", mock.Anything, "1", mock.Anything).Return(nil)

		err := s.UpdatePassword(context.Background(), "1", "oldpassword", "newpassword")
		assert.NoError(t, err)
	})
}

func TestService_DeleteUser(t *testing.T) {
	s, repo, _ := setupService()
	repo.On("DeleteUser", mock.Anything, "1").Return(nil)

	err := s.DeleteUser(context.Background(), "1")
	assert.NoError(t, err)
}

type failingReader struct{}

func (f *failingReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("entropy source failed")
}

func TestRandReader_Failures(t *testing.T) {
	origReader := randReader
	defer func() { randReader = origReader }()

	t.Run("hashPassword fails", func(t *testing.T) {
		randReader = &failingReader{}
		_, err := hashPassword("password", defaultParams)
		assert.EqualError(t, err, "entropy source failed")
	})

	t.Run("Register fails on hashPassword", func(t *testing.T) {
		s, repo, _ := setupService()
		repo.On("GetUserByEmail", mock.Anything, "test@test.com").Return(nil, errors.New("not found"))

		randReader = &failingReader{}
		_, err := s.Register(context.Background(), "Test", "test@test.com", "password")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "falha ao criptografar senha")
	})

	t.Run("UpdatePassword fails on hashPassword", func(t *testing.T) {
		randReader = origReader
		hash, _ := hashPassword("oldpass", defaultParams)

		s, repo, _ := setupService()
		repo.On("GetUserByIDWithHash", mock.Anything, "1").Return(&User{ID: "1", PasswordHash: hash}, nil)

		randReader = &failingReader{}
		err := s.UpdatePassword(context.Background(), "1", "oldpass", "newpass123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "falha ao criptografar nova senha")
	})

	t.Run("GenerateRefreshToken fails on rand.Read", func(t *testing.T) {
		s, _, _ := setupService()
		randReader = &failingReader{}
		_, err := s.GenerateRefreshToken(context.Background(), "user-1")
		assert.EqualError(t, err, "entropy source failed")
	})
}
