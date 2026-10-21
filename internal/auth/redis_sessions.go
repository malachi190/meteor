package auth

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const sessionRedisPrefix = "meteor:auth:session:"

// redisSession is the cached identity and token proof stored for an active dashboard session.
type redisSession struct {
	Account       Account `json:"account"`
	TokenHash     string  `json:"token_hash"`
	LastRenewedAt int64   `json:"last_renewed_at"`
}

// RedisSessionStore stores opaque dashboard sessions with inactivity expiry in Redis.
type RedisSessionStore struct {
	client redis.UniversalClient
}

// NewRedisSessionStore wraps an existing Redis client without taking ownership of its lifecycle.
func NewRedisSessionStore(client redis.UniversalClient) *RedisSessionStore {
	return &RedisSessionStore{client: client}
}

var createSessionScript = redis.NewScript(`
local payload = cjson.decode(ARGV[1])
local now = redis.call('TIME')
payload.last_renewed_at = tonumber(now[1])
redis.call('SET', KEYS[1], cjson.encode(payload), 'EX', ARGV[2])
return tonumber(now[1]) + tonumber(ARGV[2])
`)

var resolveSessionScript = redis.NewScript(`
local raw = redis.call('GET', KEYS[1])
if not raw then return nil end
local session = cjson.decode(raw)
if session.token_hash ~= ARGV[1] then return nil end
local now = redis.call('TIME')
local now_seconds = tonumber(now[1])
local expires_at = 0
if now_seconds - tonumber(session.last_renewed_at) >= tonumber(ARGV[2]) then
  session.last_renewed_at = now_seconds
  raw = cjson.encode(session)
  redis.call('SET', KEYS[1], raw, 'EX', ARGV[3])
  expires_at = now_seconds + tonumber(ARGV[3])
end
return {raw, expires_at}
`)

var revokeSessionScript = redis.NewScript(`
local raw = redis.call('GET', KEYS[1])
if not raw then return 0 end
local session = cjson.decode(raw)
if session.token_hash ~= ARGV[1] then return 0 end
return redis.call('DEL', KEYS[1])
`)

// Create replaces the user's current session and returns its Redis-enforced expiry time.
func (s *RedisSessionStore) Create(ctx context.Context, account Account, tokenHash []byte) (time.Time, error) {
	if !isUUID(account.TenantID) || !isUUID(account.ID) || len(tokenHash) != 32 {
		return time.Time{}, errors.New("invalid dashboard session data")
	}
	key := sessionKey(account.TenantID, account.ID)
	value, err := json.Marshal(redisSession{
		Account: account, TokenHash: hex.EncodeToString(tokenHash),
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("encode dashboard session: %w", err)
	}
	expiresAt, err := createSessionScript.Run(ctx, s.client, []string{key}, string(value), int64(sessionTTL.Seconds())).Int64()
	if err != nil {
		return time.Time{}, fmt.Errorf("cache dashboard session: %w", err)
	}
	return time.Unix(expiresAt, 0).UTC(), nil
}

// Resolve checks the token hash and renews the inactivity window only when due.
func (s *RedisSessionStore) Resolve(ctx context.Context, tenantID, userID string, tokenHash []byte) (Account, *time.Time, error) {
	if !isUUID(tenantID) || !isUUID(userID) || len(tokenHash) != 32 {
		return Account{}, nil, ErrInvalidCredentials
	}
	result, err := resolveSessionScript.Run(ctx, s.client,
		[]string{sessionKey(tenantID, userID)},
		hex.EncodeToString(tokenHash), int64(sessionRenewal.Seconds()), int64(sessionTTL.Seconds()),
	).Result()
	if errors.Is(err, redis.Nil) || result == nil {
		return Account{}, nil, ErrInvalidCredentials
	}
	if err != nil {
		return Account{}, nil, fmt.Errorf("read dashboard session cache: %w", err)
	}
	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return Account{}, nil, errors.New("dashboard session cache returned an invalid value")
	}
	serialized, ok := values[0].(string)
	if !ok {
		return Account{}, nil, errors.New("dashboard session cache returned invalid session data")
	}
	var session redisSession
	if err := json.Unmarshal([]byte(serialized), &session); err != nil {
		return Account{}, nil, fmt.Errorf("decode dashboard session cache: %w", err)
	}
	storedHash, err := hex.DecodeString(session.TokenHash)
	if err != nil || subtle.ConstantTimeCompare(storedHash, tokenHash) != 1 ||
		!sameIdentity(session.Account, tenantID, userID) {
		return Account{}, nil, ErrInvalidCredentials
	}
	newExpiry, err := redisInt64(values[1])
	if err != nil {
		return Account{}, nil, fmt.Errorf("read dashboard session expiry: %w", err)
	}
	if newExpiry == 0 {
		return session.Account, nil, nil
	}
	renewedUntil := time.Unix(newExpiry, 0).UTC()
	return session.Account, &renewedUntil, nil
}

// Revoke deletes the current session only when its stored hash matches the presented token.
func (s *RedisSessionStore) Revoke(ctx context.Context, account Account, tokenHash []byte) error {
	if !isUUID(account.TenantID) || !isUUID(account.ID) || len(tokenHash) != 32 {
		return errors.New("invalid dashboard session data")
	}
	if _, err := revokeSessionScript.Run(ctx, s.client,
		[]string{sessionKey(account.TenantID, account.ID)}, hex.EncodeToString(tokenHash),
	).Result(); err != nil {
		return fmt.Errorf("revoke dashboard session cache: %w", err)
	}
	return nil
}

// RevokeUser removes every active session for one tenant-scoped dashboard user.
func (s *RedisSessionStore) RevokeUser(ctx context.Context, tenantID, userID string) error {
	if !isUUID(tenantID) || !isUUID(userID) {
		return errors.New("invalid dashboard user identifiers")
	}
	if err := s.client.Del(ctx, sessionKey(tenantID, userID)).Err(); err != nil {
		return fmt.Errorf("revoke dashboard user sessions: %w", err)
	}
	return nil
}

// sessionKey gives each tenant and user a stable Redis key with a Redis Cluster hash tag.
func sessionKey(tenantID, userID string) string {
	return sessionRedisPrefix + "{" + tenantID + ":" + userID + "}"
}

// sameIdentity rejects cached account data whose IDs differ from the requested key.
func sameIdentity(account Account, tenantID, userID string) bool {
	return account.TenantID == tenantID && account.ID == userID
}

// redisInt64 normalizes Redis script return types across client protocol versions.
func redisInt64(value interface{}) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case string:
		return strconv.ParseInt(typed, 10, 64)
	case []byte:
		return strconv.ParseInt(string(typed), 10, 64)
	default:
		return 0, fmt.Errorf("unexpected Redis integer type %T", value)
	}
}
