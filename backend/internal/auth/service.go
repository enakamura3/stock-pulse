package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/onigiri/stock-pulse/backend/internal/config"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/argon2"
)

// UserRepository define as operações de banco de dados para a entidade de usuário.
type UserRepository interface {
	CreateUser(ctx context.Context, name, email, passwordHash string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByIDWithHash(ctx context.Context, id string) (*User, error)
	UpdateUser(ctx context.Context, id, name, email string) (*User, error)
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	DeleteUser(ctx context.Context, id string) error
}

// Service implementa a lógica de negócio de autenticação.
type Service struct {
	repo                    UserRepository
	rdb                     *redis.Client
	jwtSecret               []byte
	accessTokenTTL          time.Duration
	refreshTokenTTL         time.Duration
	refreshTokenGracePeriod time.Duration
}

// rotateRefreshTokenScript executa a rotação atômica de refresh token com suporte a Grace Period.
// Resolve race conditions de concorrência em abas paralelas e mitiga Replay Attacks reais.
var rotateRefreshTokenScript = redis.NewScript(`
-- KEYS[1] = "refresh_token:" .. oldToken
-- KEYS[2] = "rotated_token:" .. oldToken
-- KEYS[3] = "refresh_token:" .. newToken

-- ARGV[1] = candidateToken
-- ARGV[2] = ttlSeconds (ex: 43200)
-- ARGV[3] = gracePeriodMillis (ex: 30000)
-- ARGV[4] = nowUnixMilli (timestamp atual em milissegundos)
-- ARGV[5] = "used_refresh_token:" .. oldToken
-- ARGV[6] = oldToken

-- 1. Verifica se oldToken já foi rotacionado anteriormente (está em rotated_token)
local rotatedData = redis.call('HMGET', KEYS[2], 'user_id', 'new_token', 'rotated_at')
local rotatedUserID = rotatedData[1]
local existingNewToken = rotatedData[2]
local rotatedAtStr = rotatedData[3]

if rotatedUserID and rotatedUserID ~= false and existingNewToken and existingNewToken ~= false and rotatedAtStr and rotatedAtStr ~= false then
    local rotatedAt = tonumber(rotatedAtStr)
    local now = tonumber(ARGV[4])
    local gracePeriod = tonumber(ARGV[3])
    local elapsed = now - rotatedAt

    if elapsed >= 0 and elapsed <= gracePeriod then
        -- DENTRO DO GRACE PERIOD:
        -- Retorna idempotentemente o mesmo newToken já emitido
        return {"GRACE_PERIOD", rotatedUserID, existingNewToken}
    else
        -- FORA DO GRACE PERIOD: REPLAY ATTACK!
        -- Revoga todas as sessões ativas do usuário
        local userActiveKey = "user_active_tokens:" .. rotatedUserID
        local activeTokens = redis.call('SMEMBERS', userActiveKey)
        if activeTokens and #activeTokens > 0 then
            for _, t in ipairs(activeTokens) do
                redis.call('DEL', "refresh_token:" .. t)
            end
        end
        redis.call('DEL', userActiveKey)
        return {"REPLAY_ATTACK", rotatedUserID, ""}
    end
end

-- 1b. Checagem de compatibilidade com chave legada used_refresh_token
local legacyUserID = redis.call('GET', ARGV[5])
if legacyUserID and legacyUserID ~= false then
    local userActiveKey = "user_active_tokens:" .. legacyUserID
    local activeTokens = redis.call('SMEMBERS', userActiveKey)
    if activeTokens and #activeTokens > 0 then
        for _, t in ipairs(activeTokens) do
            redis.call('DEL', "refresh_token:" .. t)
        end
    end
    redis.call('DEL', userActiveKey)
    return {"REPLAY_ATTACK", legacyUserID, ""}
end

-- 2. Verifica se oldToken é um token ativo válido
local userID = redis.call('GET', KEYS[1])
if not userID or userID == false then
    return {"NOT_FOUND", "", ""}
end

-- 3. Token válido! Executa a rotação atômica:
local userActiveKey = "user_active_tokens:" .. userID
redis.call('DEL', KEYS[1])
redis.call('SREM', userActiveKey, ARGV[6])

-- Registra o novo refresh token
redis.call('SET', KEYS[3], userID, 'EX', ARGV[2])
redis.call('SADD', userActiveKey, ARGV[1])
redis.call('EXPIRE', userActiveKey, ARGV[2])

-- Registra metadados de rotação para tolerância a concorrência (Grace Period)
redis.call('HSET', KEYS[2], 'user_id', userID, 'new_token', ARGV[1], 'rotated_at', ARGV[4])
redis.call('EXPIRE', KEYS[2], ARGV[2])

return {"ROTATED", userID, ARGV[1]}
`)

// NewService cria uma nova instância de Service.
func NewService(repo UserRepository, rdb *redis.Client, jwtSecret string) *Service {
	accessTTL := config.Envs.JWTAccessTokenTTL
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}
	refreshTTL := config.Envs.JWTRefreshTokenTTL
	if refreshTTL <= 0 {
		refreshTTL = 12 * time.Hour
	}
	gracePeriod := config.Envs.JWTRefreshGracePeriod
	if gracePeriod <= 0 {
		gracePeriod = 30 * time.Second
	}

	return &Service{
		repo:                    repo,
		rdb:                     rdb,
		jwtSecret:               []byte(jwtSecret),
		accessTokenTTL:          accessTTL,
		refreshTokenTTL:         refreshTTL,
		refreshTokenGracePeriod: gracePeriod,
	}
}

// SetGracePeriod permite customizar a janela de tolerância de rotação (útil para testes).
func (s *Service) SetGracePeriod(d time.Duration) {
	s.refreshTokenGracePeriod = d
}

type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// Parâmetros recomendados pela OWASP para Argon2id.
var defaultParams = &argon2Params{
	memory:      64 * 1024, // 64 MB
	iterations:  1,
	parallelism: 4,
	saltLength:  16,
	keyLength:   32,
}

var randReader io.Reader = rand.Reader

// hashPassword gera um hash seguro usando Argon2id no formato padrão.
func hashPassword(password string, params *argon2Params) (string, error) {
	salt := make([]byte, params.saltLength)
	if _, err := io.ReadFull(randReader, salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, params.keyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		params.memory, params.iterations, params.parallelism, b64Salt, b64Hash)

	return encoded, nil
}

// comparePasswordAndHash verifica se uma senha candidata corresponde ao hash codificado.
func comparePasswordAndHash(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("formato de hash inválido")
	}

	var memory, iterations uint32
	var parallelism uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	keyLength := uint32(len(decodedHash))

	comparisonHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)

	if subtle.ConstantTimeCompare(decodedHash, comparisonHash) == 1 {
		return true, nil
	}

	return false, nil
}

// Register cria um novo usuário no banco com senha criptografada em Argon2id.
func (s *Service) Register(ctx context.Context, name, email, password string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)

	if email == "" || name == "" {
		return nil, errors.New("todos os campos são obrigatórios")
	}
	if len(password) < 6 {
		return nil, errors.New("a senha deve ter no mínimo 6 caracteres")
	}

	// Verifica duplicidade no banco
	existing, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, errors.New("este e-mail já está cadastrado")
	}

	hash, err := hashPassword(password, defaultParams)
	if err != nil {
		return nil, fmt.Errorf("falha ao criptografar senha: %w", err)
	}

	return s.repo.CreateUser(ctx, name, email, hash)
}

// Login valida o e-mail/senha e retorna o usuário logado e os tokens gerados.
func (s *Service) Login(ctx context.Context, email, password string) (*User, string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, "", "", errors.New("e-mail ou senha incorretos")
	}

	match, err := comparePasswordAndHash(password, user.PasswordHash)
	if err != nil || !match {
		return nil, "", "", errors.New("e-mail ou senha incorretos")
	}

	accessToken, err := s.GenerateAccessToken(user)
	if err != nil {
		return nil, "", "", fmt.Errorf("falha ao gerar access token: %w", err)
	}

	refreshToken, err := s.GenerateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("falha ao gerar refresh token: %w", err)
	}

	return user, accessToken, refreshToken, nil
}

// GenerateAccessToken gera um JWT Access Token assinado com a validade configurada.
func (s *Service) GenerateAccessToken(user *User) (string, error) {
	if user == nil {
		return "", errors.New("usuário não fornecido")
	}
	if len(s.jwtSecret) == 0 {
		return "", errors.New("jwtSecret não configurado")
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(s.accessTokenTTL).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *Service) generateRandomToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := io.ReadFull(randReader, tokenBytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}

// GenerateRefreshToken cria um token seguro e armazena no Redis com o TTL configurado,
// registrando-o também no conjunto de tokens ativos do usuário para controle de sessão.
func (s *Service) GenerateRefreshToken(ctx context.Context, userID string) (string, error) {
	refreshToken, err := s.generateRandomToken()
	if err != nil {
		return "", err
	}

	// Chave com prefixo para fácil identificação
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	err = s.rdb.Set(ctx, key, userID, s.refreshTokenTTL).Err()
	if err != nil {
		return "", err
	}

	activeTokensKey := fmt.Sprintf("user_active_tokens:%s", userID)
	if err := s.rdb.SAdd(ctx, activeTokensKey, refreshToken).Err(); err != nil {
		return "", err
	}
	if err := s.rdb.Expire(ctx, activeTokensKey, s.refreshTokenTTL).Err(); err != nil {
		return "", err
	}

	return refreshToken, nil
}

// ValidateRefreshToken resgata o ID do usuário no Redis associado ao refresh token.
func (s *Service) ValidateRefreshToken(ctx context.Context, token string) (string, error) {
	key := fmt.Sprintf("refresh_token:%s", token)
	userID, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", errors.New("sessão expirada ou inválida")
		}
		return "", err
	}
	return userID, nil
}

// RotateRefreshToken consome um refresh token existente e gera um novo par de tokens (Refresh Token Rotation).
// A operação é executada atomicamente no Redis com suporte a Grace Period (tolerância a concorrência de abas).
// Se o token for reutilizado fora da janela de tolerância, revoga todas as sessões ativas do usuário para mitigar replay attacks.
func (s *Service) RotateRefreshToken(ctx context.Context, oldToken string) (string, string, error) {
	oldToken = strings.TrimSpace(oldToken)
	if oldToken == "" {
		return "", "", errors.New("token inválido")
	}

	if s.rdb == nil {
		return "", "", errors.New("cliente redis não inicializado")
	}

	candidateToken, err := s.generateRandomToken()
	if err != nil {
		return "", "", fmt.Errorf("falha ao gerar novo refresh token: %w", err)
	}

	keys := []string{
		fmt.Sprintf("refresh_token:%s", oldToken),
		fmt.Sprintf("rotated_token:%s", oldToken),
		fmt.Sprintf("refresh_token:%s", candidateToken),
	}
	args := []interface{}{
		candidateToken,
		int64(s.refreshTokenTTL.Seconds()),
		int64(s.refreshTokenGracePeriod.Milliseconds()),
		time.Now().UnixMilli(),
		fmt.Sprintf("used_refresh_token:%s", oldToken),
		oldToken,
	}

	rawResult, err := rotateRefreshTokenScript.Run(ctx, s.rdb, keys, args...).Slice()
	if err != nil {
		return "", "", err
	}

	return parseRotateResult(rawResult)
}

func parseRotateResult(rawResult []interface{}) (string, string, error) {
	if len(rawResult) < 3 {
		return "", "", errors.New("resposta inesperada do script de rotação de token")
	}

	status, _ := rawResult[0].(string)
	userID, _ := rawResult[1].(string)
	newToken, _ := rawResult[2].(string)

	switch status {
	case "ROTATED":
		return userID, newToken, nil
	case "GRACE_PERIOD":
		slog.Info("Refresh token reutilizado dentro da janela de tolerância (Grace Period)", "user_id", userID)
		return userID, newToken, nil
	case "REPLAY_ATTACK":
		slog.Warn("Tentativa de reutilização de refresh token detectada (Replay Attack)", "user_id", userID)
		return "", "", errors.New("tentativa de reutilização de token detectada")
	case "NOT_FOUND":
		return "", "", errors.New("sessão expirada ou inválida")
	default:
		return "", "", errors.New("status desconhecido na rotação de token")
	}
}

// RevokeRefreshToken invalida a sessão apagando o refresh token do Redis e desvinculando-o das sessões ativas.
func (s *Service) RevokeRefreshToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("refresh_token:%s", token)
	rotatedKey := fmt.Sprintf("rotated_token:%s", token)
	usedKey := fmt.Sprintf("used_refresh_token:%s", token)

	userID, err := s.rdb.Get(ctx, key).Result()
	if err == nil && userID != "" {
		_ = s.rdb.SRem(ctx, fmt.Sprintf("user_active_tokens:%s", userID), token).Err()
	}
	_ = s.rdb.Del(ctx, usedKey).Err()
	_ = s.rdb.Del(ctx, rotatedKey).Err()
	return s.rdb.Del(ctx, key).Err()
}

// GetUserByID retorna um usuário pelo ID.
func (s *Service) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.repo.GetUserByID(ctx, id)
}

// UpdateProfile atualiza o nome e e-mail do usuário.
func (s *Service) UpdateProfile(ctx context.Context, id, name, email string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)

	if email == "" || name == "" {
		return nil, errors.New("todos os campos são obrigatórios")
	}

	// Verifica se o e-mail já está em uso por outro usuário
	existing, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil && existing != nil && existing.ID != id {
		return nil, errors.New("este e-mail já está cadastrado por outro usuário")
	}

	return s.repo.UpdateUser(ctx, id, name, email)
}

// UpdatePassword valida a senha atual e define a nova senha.
func (s *Service) UpdatePassword(ctx context.Context, id, currentPassword, newPassword string) error {
	if len(newPassword) < 6 {
		return errors.New("a nova senha deve ter no mínimo 6 caracteres")
	}

	user, err := s.repo.GetUserByIDWithHash(ctx, id)
	if err != nil {
		return errors.New("usuário não encontrado")
	}

	match, err := comparePasswordAndHash(currentPassword, user.PasswordHash)
	if err != nil || !match {
		return errors.New("senha atual incorreta")
	}

	hash, err := hashPassword(newPassword, defaultParams)
	if err != nil {
		return fmt.Errorf("falha ao criptografar nova senha: %w", err)
	}

	return s.repo.UpdatePassword(ctx, id, hash)
}

// DeleteUser deleta o usuário do banco de dados.
func (s *Service) DeleteUser(ctx context.Context, id string) error {
	return s.repo.DeleteUser(ctx, id)
}
