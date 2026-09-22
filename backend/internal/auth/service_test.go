package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redismock/v9"
	"github.com/golang-jwt/jwt/v5"
	"github.com/onigiri/stock-pulse/backend/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func setupMiniRedisService(t *testing.T) (*Service, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	repoMock := new(MockUserRepository)
	s := NewService(repoMock, rdb, "secret")
	return s, mr
}

func TestService_RotateRefreshToken(t *testing.T) {
	t.Run("Empty token", func(t *testing.T) {
		s, _, _ := setupService()
		_, _, err := s.RotateRefreshToken(context.Background(), "   ")
		assert.EqualError(t, err, "token inválido")
	})

	t.Run("Nil Redis client", func(t *testing.T) {
		repoMock := new(MockUserRepository)
		s := NewService(repoMock, nil, "secret")
		_, _, err := s.RotateRefreshToken(context.Background(), "valid_token")
		assert.EqualError(t, err, "cliente redis não inicializado")
	})

	t.Run("Non-existent or expired token", func(t *testing.T) {
		s, _ := setupMiniRedisService(t)
		_, _, err := s.RotateRefreshToken(context.Background(), "non_existent_token")
		assert.EqualError(t, err, "sessão expirada ou inválida")
	})

	t.Run("Success atomic rotation", func(t *testing.T) {
		s, mr := setupMiniRedisService(t)
		ctx := context.Background()

		oldToken, err := s.GenerateRefreshToken(ctx, "user-1")
		require.NoError(t, err)

		userID, newToken, err := s.RotateRefreshToken(ctx, oldToken)
		assert.NoError(t, err)
		assert.Equal(t, "user-1", userID)
		assert.NotEmpty(t, newToken)
		assert.NotEqual(t, oldToken, newToken)

		// Token antigo não é mais ativo
		assert.False(t, mr.Exists("refresh_token:"+oldToken))
		// Novo token é ativo
		assert.True(t, mr.Exists("refresh_token:"+newToken))
		// rotated_token foi registrado
		assert.True(t, mr.Exists("rotated_token:"+oldToken))
		// active tokens contém apenas o novo token
		activeTokens, _ := s.rdb.SMembers(ctx, "user_active_tokens:user-1").Result()
		assert.Equal(t, []string{newToken}, activeTokens)
	})

	t.Run("Grace Period allows retry with old token", func(t *testing.T) {
		s, _ := setupMiniRedisService(t)
		ctx := context.Background()

		oldToken, err := s.GenerateRefreshToken(ctx, "user-1")
		require.NoError(t, err)

		// 1ª rotação
		userID1, newToken1, err := s.RotateRefreshToken(ctx, oldToken)
		require.NoError(t, err)

		// 2ª rotação com o mesmo oldToken (dentro do grace period de 30s)
		userID2, newToken2, err := s.RotateRefreshToken(ctx, oldToken)
		assert.NoError(t, err)
		assert.Equal(t, userID1, userID2)
		assert.Equal(t, newToken1, newToken2)

		// Sessão permanece válida
		validUser, err := s.ValidateRefreshToken(ctx, newToken1)
		assert.NoError(t, err)
		assert.Equal(t, "user-1", validUser)
	})

	t.Run("Concurrent requests within grace window", func(t *testing.T) {
		s, _ := setupMiniRedisService(t)
		ctx := context.Background()

		oldToken, err := s.GenerateRefreshToken(ctx, "user-concurrent")
		require.NoError(t, err)

		concurrency := 20
		results := make([]string, concurrency)
		errorsList := make([]error, concurrency)
		var wg sync.WaitGroup
		start := make(chan struct{})

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				<-start
				_, newToken, rErr := s.RotateRefreshToken(ctx, oldToken)
				results[idx] = newToken
				errorsList[idx] = rErr
			}(i)
		}
		close(start)
		wg.Wait()

		for i := 0; i < concurrency; i++ {
			assert.NoError(t, errorsList[i], "request %d failed", i)
			assert.NotEmpty(t, results[i])
			assert.Equal(t, results[0], results[i], "todas as requisições concorrentes devem receber o mesmo token")
		}

		// Apenas 1 token ativo no Redis
		activeTokens, err := s.rdb.SMembers(ctx, "user_active_tokens:user-concurrent").Result()
		assert.NoError(t, err)
		assert.Len(t, activeTokens, 1)
	})

	t.Run("Replay attack detected after grace period expires", func(t *testing.T) {
		s, mr := setupMiniRedisService(t)
		s.SetGracePeriod(100 * time.Millisecond) // Grace period de 100ms para teste rápido
		ctx := context.Background()

		oldToken, err := s.GenerateRefreshToken(ctx, "user-replay")
		require.NoError(t, err)

		// Rotação inicial
		_, newToken, err := s.RotateRefreshToken(ctx, oldToken)
		require.NoError(t, err)

		// Aguarda término do grace period (150ms > 100ms)
		time.Sleep(150 * time.Millisecond)

		// Atacante tenta reutilizar oldToken após o grace period
		userID, replayedToken, err := s.RotateRefreshToken(ctx, oldToken)
		assert.Empty(t, userID)
		assert.Empty(t, replayedToken)
		assert.EqualError(t, err, "tentativa de reutilização de token detectada")

		// Todas as sessões do usuário devem ter sido revogadas
		assert.False(t, mr.Exists("refresh_token:"+newToken))
		assert.False(t, mr.Exists("user_active_tokens:user-replay"))
	})

	t.Run("Legacy used_refresh_token triggers replay revocation", func(t *testing.T) {
		s, mr := setupMiniRedisService(t)
		ctx := context.Background()

		// Cria uma sessão ativa para o usuário
		activeToken, err := s.GenerateRefreshToken(ctx, "user-legacy")
		require.NoError(t, err)

		// Simula chave legada no Redis
		mr.Set("used_refresh_token:legacy_token", "user-legacy")

		// Replay com token legado
		_, _, err = s.RotateRefreshToken(ctx, "legacy_token")
		assert.EqualError(t, err, "tentativa de reutilização de token detectada")

		// Sessão ativa foi revogada
		assert.False(t, mr.Exists("refresh_token:"+activeToken))
		assert.False(t, mr.Exists("user_active_tokens:user-legacy"))
	})

	t.Run("Redis script execution error", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		s := NewService(new(MockUserRepository), rdb, "secret")
		mr.Close() // Fecha o servidor Redis para forçar falha no comando

		_, _, err := s.RotateRefreshToken(context.Background(), "token")
		assert.Error(t, err)
	})

	t.Run("Script returns malformed or unexpected slice", func(t *testing.T) {
		_, _, err := parseRotateResult([]interface{}{"SHORT"})
		assert.EqualError(t, err, "resposta inesperada do script de rotação de token")
	})

	t.Run("Script returns unknown status", func(t *testing.T) {
		_, _, err := parseRotateResult([]interface{}{"UNKNOWN_STATUS", "u1", "t1"})
		assert.EqualError(t, err, "status desconhecido na rotação de token")
	})
}

func TestService_RevokeRefreshToken(t *testing.T) {
	t.Run("Success with user in token", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("refresh_token:token").SetVal("user-1")
		rdbMock.ExpectSRem("user_active_tokens:user-1", "token").SetVal(1)
		rdbMock.ExpectDel("used_refresh_token:token").SetVal(1)
		rdbMock.ExpectDel("rotated_token:token").SetVal(1)
		rdbMock.ExpectDel("refresh_token:token").SetVal(1)

		err := s.RevokeRefreshToken(context.Background(), "token")
		assert.NoError(t, err)
	})

	t.Run("Token not found in redis", func(t *testing.T) {
		s, _, rdbMock := setupService()
		rdbMock.ExpectGet("refresh_token:token").SetErr(redis.Nil)
		rdbMock.ExpectDel("used_refresh_token:token").SetVal(0)
		rdbMock.ExpectDel("rotated_token:token").SetVal(0)
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

	t.Run("RotateRefreshToken fails on rand.Read", func(t *testing.T) {
		s, _, _ := setupService()
		randReader = &failingReader{}
		_, _, err := s.RotateRefreshToken(context.Background(), "valid-token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "falha ao gerar novo refresh token")
	})
}
