package telegram

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gopkg.in/telebot.v3"
)

type mockBotController struct {
	mu           sync.Mutex
	startCalled  int
	stopCalled   int
	blockOnStart bool
	stopCh       chan struct{}
}

func newMockBotController(blockOnStart bool) *mockBotController {
	return &mockBotController{
		blockOnStart: blockOnStart,
		stopCh:       make(chan struct{}),
	}
}

func (m *mockBotController) Start() {
	m.mu.Lock()
	m.startCalled++
	block := m.blockOnStart
	m.mu.Unlock()

	if block {
		<-m.stopCh
	}
}

func (m *mockBotController) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCalled++
	select {
	case <-m.stopCh:
	default:
		close(m.stopCh)
	}
}

func TestNewLeaderElector(t *testing.T) {
	t.Run("Default configuration values with real hostname", func(t *testing.T) {
		bot := newMockBotController(false)
		elector := NewLeaderElector(bot, nil, LeaderElectorConfig{})

		cfg := elector.Config()
		assert.Equal(t, DefaultLeaderLockKey, cfg.LockKey)
		assert.Equal(t, DefaultLeaderLockTTL, cfg.LockTTL)
		assert.Equal(t, DefaultRenewInterval, cfg.RenewInterval)
		assert.Equal(t, DefaultRetryInterval, cfg.RetryInterval)
		assert.NotEmpty(t, cfg.InstanceID)
		assert.False(t, elector.IsLeader())
	})

	t.Run("Default configuration values when hostname fails", func(t *testing.T) {
		orig := hostnameFunc
		defer func() { hostnameFunc = orig }()
		hostnameFunc = func() (string, error) {
			return "", errors.New("hostname lookup error")
		}

		bot := newMockBotController(false)
		elector := NewLeaderElector(bot, nil, LeaderElectorConfig{})
		assert.Contains(t, elector.Config().InstanceID, "instance-")
	})

	t.Run("Custom configuration values", func(t *testing.T) {
		bot := newMockBotController(false)
		customCfg := LeaderElectorConfig{
			LockKey:       "custom:lock:key",
			InstanceID:    "node-custom-1",
			LockTTL:       15 * time.Second,
			RenewInterval: 3 * time.Second,
			RetryInterval: 1 * time.Second,
		}
		elector := NewLeaderElector(bot, nil, customCfg)

		cfg := elector.Config()
		assert.Equal(t, "custom:lock:key", cfg.LockKey)
		assert.Equal(t, "node-custom-1", cfg.InstanceID)
		assert.Equal(t, 15*time.Second, cfg.LockTTL)
		assert.Equal(t, 3*time.Second, cfg.RenewInterval)
		assert.Equal(t, 1*time.Second, cfg.RetryInterval)
	})
}

func TestLeaderElector_IsLeader_SetLeader(t *testing.T) {
	bot := newMockBotController(false)
	elector := NewLeaderElector(bot, nil, LeaderElectorConfig{})

	assert.False(t, elector.IsLeader())
	elector.setLeader(true)
	assert.True(t, elector.IsLeader())
	elector.setLeader(false)
	assert.False(t, elector.IsLeader())
}

func TestLeaderElector_NilRedis(t *testing.T) {
	bot := newMockBotController(false)
	elector := NewLeaderElector(bot, nil, LeaderElectorConfig{})

	ctx := context.Background()

	acquired, err := elector.Acquire(ctx)
	assert.True(t, acquired)
	assert.NoError(t, err)

	renewed, err := elector.Renew(ctx)
	assert.True(t, renewed)
	assert.NoError(t, err)

	err = elector.Release(ctx)
	assert.NoError(t, err)

	err = elector.Run(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, bot.startCalled)
}

func TestLeaderElector_NilBot(t *testing.T) {
	elector := NewLeaderElector(nil, nil, LeaderElectorConfig{})
	err := elector.Run(context.Background())
	assert.NoError(t, err)
}

func TestLeaderElector_AcquireRenewRelease(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	bot1 := newMockBotController(false)
	elector1 := NewLeaderElector(bot1, rdb, LeaderElectorConfig{
		LockKey:    "test:lock:poller",
		InstanceID: "inst-1",
		LockTTL:    5 * time.Second,
	})

	bot2 := newMockBotController(false)
	elector2 := NewLeaderElector(bot2, rdb, LeaderElectorConfig{
		LockKey:    "test:lock:poller",
		InstanceID: "inst-2",
		LockTTL:    500 * time.Millisecond, // triggers ttlSec <= 0 fallback in Renew
	})

	ctx := context.Background()

	// 1. Elector 1 acquires lock
	ok, err := elector1.Acquire(ctx)
	assert.NoError(t, err)
	assert.True(t, ok)

	val, err := mr.Get("test:lock:poller")
	assert.NoError(t, err)
	assert.Equal(t, "inst-1", val)

	// 2. Elector 2 tries to acquire same lock and fails
	ok2, err2 := elector2.Acquire(ctx)
	assert.NoError(t, err2)
	assert.False(t, ok2)

	// 3. Elector 1 renews successfully
	renewed1, err := elector1.Renew(ctx)
	assert.NoError(t, err)
	assert.True(t, renewed1)

	// 4. Elector 2 tries to renew and fails because it does not own the lock
	renewed2, err := elector2.Renew(ctx)
	assert.NoError(t, err)
	assert.False(t, renewed2)

	// 5. Elector 2 tries to release and does nothing
	err = elector2.Release(ctx)
	assert.NoError(t, err)
	assert.True(t, mr.Exists("test:lock:poller"))

	// 6. Elector 1 releases lock
	err = elector1.Release(ctx)
	assert.NoError(t, err)
	assert.False(t, mr.Exists("test:lock:poller"))
}

func TestLeaderElector_RedisErrors(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	bot := newMockBotController(false)
	elector := NewLeaderElector(bot, rdb, LeaderElectorConfig{
		LockKey:    "test:lock:err",
		InstanceID: "inst-err",
	})

	// Close miniredis to simulate connection failure
	mr.Close()
	rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := elector.Renew(ctx)
	assert.Error(t, err)

	err = elector.Release(ctx)
	assert.Error(t, err)
}

func TestLeaderElector_Run_ContextAlreadyCancelled(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	bot := newMockBotController(false)
	elector := NewLeaderElector(bot, rdb, LeaderElectorConfig{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := elector.Run(ctx)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestLeaderElector_Run_AcquireError_ThenCancel(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	bot := newMockBotController(false)
	elector := NewLeaderElector(bot, rdb, LeaderElectorConfig{
		RetryInterval: 10 * time.Millisecond,
	})

	// Close server to force acquire error
	mr.Close()
	rdb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := elector.Run(ctx)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
}

func TestLeaderElector_Run_SingleInstance_GracefulShutdown(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	bot := newMockBotController(true)
	elector := NewLeaderElector(bot, rdb, LeaderElectorConfig{
		LockKey:       "test:lock:lifecycle",
		InstanceID:    "inst-single",
		LockTTL:       2 * time.Second,
		RenewInterval: 50 * time.Millisecond,
		RetryInterval: 50 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- elector.Run(ctx)
	}()

	// Wait for instance to become leader
	assert.Eventually(t, func() bool {
		return elector.IsLeader()
	}, 1*time.Second, 20*time.Millisecond)

	assert.Equal(t, 1, bot.startCalled)
	assert.True(t, mr.Exists("test:lock:lifecycle"))

	// Let it run through at least one renewal
	time.Sleep(120 * time.Millisecond)

	// Cancel context to simulate graceful shutdown
	cancel()

	err := <-errCh
	assert.True(t, errors.Is(err, context.Canceled))
	assert.False(t, elector.IsLeader())
	assert.Equal(t, 1, bot.stopCalled)
	assert.False(t, mr.Exists("test:lock:lifecycle"))
}

func TestLeaderElector_Run_StandbyAndFailover(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	lockKey := "test:lock:failover"
	// Pre-set lock with old leader
	_ = mr.Set(lockKey, "leader-old")

	bot := newMockBotController(true)
	elector := NewLeaderElector(bot, rdb, LeaderElectorConfig{
		LockKey:       lockKey,
		InstanceID:    "leader-new",
		LockTTL:       2 * time.Second,
		RenewInterval: 50 * time.Millisecond,
		RetryInterval: 40 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- elector.Run(ctx)
	}()

	// Verify it starts in standby because old leader holds the lock
	time.Sleep(100 * time.Millisecond)
	assert.False(t, elector.IsLeader())
	assert.Equal(t, 0, bot.startCalled)

	// Old leader drops lock
	mr.Del(lockKey)

	// Elector should acquire lock and become leader
	assert.Eventually(t, func() bool {
		return elector.IsLeader()
	}, 1*time.Second, 20*time.Millisecond)

	assert.Equal(t, 1, bot.startCalled)
	val, err := mr.Get(lockKey)
	assert.NoError(t, err)
	assert.Equal(t, "leader-new", val)

	cancel()
	<-errCh
}

func TestLeaderElector_Run_LostLeadership(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	lockKey := "test:lock:lost"
	bot := newMockBotController(true)
	elector := NewLeaderElector(bot, rdb, LeaderElectorConfig{
		LockKey:       lockKey,
		InstanceID:    "inst-lost",
		LockTTL:       2 * time.Second,
		RenewInterval: 50 * time.Millisecond,
		RetryInterval: 50 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- elector.Run(ctx)
	}()

	// Wait until it becomes leader
	assert.Eventually(t, func() bool {
		return elector.IsLeader()
	}, 1*time.Second, 20*time.Millisecond)

	// Overwrite lock with another instance ID to simulate lease theft/loss
	_ = mr.Set(lockKey, "intruder-id")

	// Elector should detect lost leadership on next renew and stop the bot
	assert.Eventually(t, func() bool {
		return !elector.IsLeader()
	}, 1*time.Second, 20*time.Millisecond)

	assert.Equal(t, 1, bot.stopCalled)

	cancel()
	<-errCh
}

func TestLeaderElector_Run_BotStopsUnexpectedly(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	lockKey := "test:lock:botstop"
	// Bot that does NOT block on start
	bot := newMockBotController(false)
	elector := NewLeaderElector(bot, rdb, LeaderElectorConfig{
		LockKey:       lockKey,
		InstanceID:    "inst-botstop",
		LockTTL:       2 * time.Second,
		RenewInterval: 50 * time.Millisecond,
		RetryInterval: 50 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- elector.Run(ctx)
	}()

	// Bot starts and exits immediately -> runLeaderLoop catches botStopped
	time.Sleep(100 * time.Millisecond)

	cancel()
	<-errCh
}

func TestBotRunner_StartWithLeaderElection(t *testing.T) {
	t.Run("nil BotRunner", func(t *testing.T) {
		var r *BotRunner
		err := r.StartWithLeaderElection(context.Background(), nil)
		assert.NoError(t, err)
	})

	t.Run("BotRunner with nil bot", func(t *testing.T) {
		r := &BotRunner{bot: nil}
		err := r.StartWithLeaderElection(context.Background(), nil)
		assert.NoError(t, err)
	})

	t.Run("BotRunner with miniredis success", func(t *testing.T) {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		defer rdb.Close()

		b, err := telebot.NewBot(telebot.Settings{Offline: true})
		assert.NoError(t, err)

		runner := &BotRunner{bot: b}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err = runner.StartWithLeaderElection(ctx, rdb, LeaderElectorConfig{
			LockKey:       "test:botrunner:election",
			RenewInterval: 30 * time.Millisecond,
			RetryInterval: 30 * time.Millisecond,
		})
		assert.True(t, errors.Is(err, context.Canceled))
	})
}
