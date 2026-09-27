package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Constantes de configuração padrão para a eleição de liderança.
const (
	DefaultLeaderLockKey = "lock:telegram_bot_poller"
	DefaultLeaderLockTTL = 30 * time.Second
	DefaultRenewInterval = 10 * time.Second
	DefaultRetryInterval = 5 * time.Second
)

var hostnameFunc = os.Hostname

var renewLeaderScript = redis.NewScript(`
local current = redis.call('GET', KEYS[1])
if current == ARGV[1] then
    redis.call('EXPIRE', KEYS[1], tonumber(ARGV[2]))
    return 1
else
    return 0
end
`)

var releaseLeaderScript = redis.NewScript(`
local current = redis.call('GET', KEYS[1])
if current == ARGV[1] then
    return redis.call('DEL', KEYS[1])
else
    return 0
end
`)

// BotController define os métodos mínimos necessários para iniciar e parar o polling do bot.
type BotController interface {
	Start()
	Stop()
}

// LeaderElectorConfig armazena os parâmetros de eleição de liderança via Redis.
type LeaderElectorConfig struct {
	LockKey       string
	InstanceID    string
	LockTTL       time.Duration
	RenewInterval time.Duration
	RetryInterval time.Duration
}

// LeaderElector gerencia a eleição de liderança no cluster Redis para garantir
// que apenas uma réplica execute o LongPoller do Telegram bot por vez.
type LeaderElector struct {
	bot      BotController
	rdb      *redis.Client
	cfg      LeaderElectorConfig
	isLeader bool
	mu       sync.RWMutex
}

// NewLeaderElector instancia um novo gerenciador de eleição de liderança.
func NewLeaderElector(bot BotController, rdb *redis.Client, cfg LeaderElectorConfig) *LeaderElector {
	if cfg.LockKey == "" {
		cfg.LockKey = DefaultLeaderLockKey
	}
	if cfg.LockTTL <= 0 {
		cfg.LockTTL = DefaultLeaderLockTTL
	}
	if cfg.RenewInterval <= 0 {
		cfg.RenewInterval = DefaultRenewInterval
	}
	if cfg.RetryInterval <= 0 {
		cfg.RetryInterval = DefaultRetryInterval
	}
	if cfg.InstanceID == "" {
		host, err := hostnameFunc()
		if err != nil || host == "" {
			host = "instance"
		}
		cfg.InstanceID = fmt.Sprintf("%s-%s", host, uuid.New().String())
	}

	return &LeaderElector{
		bot: bot,
		rdb: rdb,
		cfg: cfg,
	}
}

// Config retorna uma cópia da configuração atual do elector.
func (le *LeaderElector) Config() LeaderElectorConfig {
	return le.cfg
}

// IsLeader retorna se esta instância é atualmente a líder eleita.
func (le *LeaderElector) IsLeader() bool {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.isLeader
}

func (le *LeaderElector) setLeader(val bool) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.isLeader = val
}

// Acquire tenta adquirir o lock de liderança de forma atômica no Redis.
func (le *LeaderElector) Acquire(ctx context.Context) (bool, error) {
	if le.rdb == nil {
		return true, nil
	}
	return le.rdb.SetNX(ctx, le.cfg.LockKey, le.cfg.InstanceID, le.cfg.LockTTL).Result()
}

// Renew estende o TTL do lock no Redis se e somente se esta instância ainda for a detentora.
func (le *LeaderElector) Renew(ctx context.Context) (bool, error) {
	if le.rdb == nil {
		return true, nil
	}
	ttlSec := int(le.cfg.LockTTL.Seconds())
	if ttlSec <= 0 {
		ttlSec = 1
	}
	res, err := renewLeaderScript.Run(ctx, le.rdb, []string{le.cfg.LockKey}, le.cfg.InstanceID, ttlSec).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// Release libera voluntariamente o lock no Redis se e somente se esta instância ainda for a detentora.
func (le *LeaderElector) Release(ctx context.Context) error {
	if le.rdb == nil {
		return nil
	}
	_, err := releaseLeaderScript.Run(ctx, le.rdb, []string{le.cfg.LockKey}, le.cfg.InstanceID).Result()
	return err
}

// Run executa o loop de eleição de liderança até que o contexto seja cancelado.
func (le *LeaderElector) Run(ctx context.Context) error {
	if le.bot == nil {
		return nil
	}
	if le.rdb == nil {
		slog.Warn("Redis não configurado para Leader Election do Telegram bot. Iniciando em modo standalone.")
		le.setLeader(true)
		le.bot.Start()
		return nil
	}

	slog.Info("Iniciando Leader Election para Telegram Bot Poller...",
		"instanceID", le.cfg.InstanceID,
		"lockKey", le.cfg.LockKey,
	)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		acquired, err := le.Acquire(ctx)
		if err != nil {
			slog.Warn("Erro ao tentar adquirir liderança do Telegram bot", "error", err)
		}

		if acquired {
			slog.Info("🏆 Instância eleita líder do Telegram Bot Poller! Ativando bot...", "instanceID", le.cfg.InstanceID)
			le.setLeader(true)

			botStopped := make(chan struct{})
			go func() {
				le.bot.Start()
				close(botStopped)
			}()

			lostLeadership := le.runLeaderLoop(ctx, botStopped)
			le.setLeader(false)
			le.bot.Stop()
			<-botStopped

			if !lostLeadership {
				relCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				_ = le.Release(relCtx)
				cancel()
				return ctx.Err()
			}
			slog.Warn("Liderança do Telegram bot perdida ou cedida. Retornando ao modo standby...", "instanceID", le.cfg.InstanceID)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(le.cfg.RetryInterval):
		}
	}
}

func (le *LeaderElector) runLeaderLoop(ctx context.Context, botStopped <-chan struct{}) bool {
	ticker := time.NewTicker(le.cfg.RenewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-botStopped:
			return false
		case <-ticker.C:
			renewed, err := le.Renew(ctx)
			if err != nil || !renewed {
				slog.Warn("Falha ao renovar lease de liderança do Telegram bot", "error", err, "renewed", renewed)
				return true
			}
		}
	}
}
