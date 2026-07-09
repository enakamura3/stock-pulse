package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

// Service lida com regras de negócio de autenticação, hashing e controle de sessão.
type Service struct {
	repo      UserRepository
	rdb       *redis.Client
	jwtSecret []byte
}

// NewService cria uma nova instância de Service.
func NewService(repo UserRepository, rdb *redis.Client, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		rdb:       rdb,
		jwtSecret: []byte(jwtSecret),
	}
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

// hashPassword gera um hash seguro usando Argon2id no formato padrão.
func hashPassword(password string, params *argon2Params) (string, error) {
	salt := make([]byte, params.saltLength)
	if _, err := rand.Read(salt); err != nil {
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

// GenerateAccessToken gera um JWT Access Token assinado com validade de 2 horas.
func (s *Service) GenerateAccessToken(user *User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(2 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// GenerateRefreshToken cria um token seguro e armazena no Redis com TTL de 7 dias.
func (s *Service) GenerateRefreshToken(ctx context.Context, userID string) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	refreshToken := base64.RawURLEncoding.EncodeToString(tokenBytes)

	// Chave com prefixo para fácil identificação
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	err := s.rdb.Set(ctx, key, userID, 7*24*time.Hour).Err()
	if err != nil {
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

// RevokeRefreshToken invalida a sessão apagando o refresh token do Redis.
func (s *Service) RevokeRefreshToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("refresh_token:%s", token)
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
