package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// MockRedisClient is a mock implementation of Redis client for unit tests
type MockRedisClient struct {
	mu       sync.RWMutex
	data     map[string]string
	expiries map[string]time.Time
	lists    map[string][]string
	sets     map[string]map[string]struct{}
	hashes   map[string]map[string]string
	zsets    map[string]map[string]float64

	// Callbacks for testing
	OnSet     func(key, value string, expiration time.Duration) error
	OnGet     func(key string) (string, error)
	OnDel     func(keys ...string) error
	OnSetNX   func(key, value string, expiration time.Duration) (bool, error)
	OnExpire  func(key string, expiration time.Duration) error
	OnLPush   func(key string, values ...interface{}) error
	OnBRPop   func(timeout time.Duration, keys ...string) ([]string, error)
	OnHSet    func(key string, values ...interface{}) error
	OnHGet    func(key, field string) (string, error)
	OnHGetAll func(key string) (map[string]string, error)
	OnZAdd    func(key string, members ...redis.Z) error
	OnZRangeByScore func(key string, opt *redis.ZRangeBy) ([]string, error)
}

// NewMockRedisClient creates a new mock Redis client
func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data:     make(map[string]string),
		expiries: make(map[string]time.Time),
		lists:    make(map[string][]string),
		sets:     make(map[string]map[string]struct{}),
		hashes:   make(map[string]map[string]string),
		zsets:    make(map[string]map[string]float64),
	}
}

// Set sets a key-value pair
func (m *MockRedisClient) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	if m.OnSet != nil {
		return m.OnSet(key, value, expiration)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	if expiration > 0 {
		m.expiries[key] = time.Now().Add(expiration)
	}
	return nil
}

// Get gets a value by key
func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	if m.OnGet != nil {
		return m.OnGet(key)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check expiry
	if expiry, ok := m.expiries[key]; ok && time.Now().After(expiry) {
		delete(m.data, key)
		delete(m.expiries, key)
		return "", redis.Nil
	}

	val, ok := m.data[key]
	if !ok {
		return "", redis.Nil
	}
	return val, nil
}

// Del deletes keys
func (m *MockRedisClient) Del(ctx context.Context, keys ...string) error {
	if m.OnDel != nil {
		return m.OnDel(keys...)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.data, key)
		delete(m.expiries, key)
		delete(m.lists, key)
		delete(m.hashes, key)
		delete(m.zsets, key)
	}
	return nil
}

// SetNX sets a key only if it doesn't exist
func (m *MockRedisClient) SetNX(ctx context.Context, key, value string, expiration time.Duration) (bool, error) {
	if m.OnSetNX != nil {
		return m.OnSetNX(key, value, expiration)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if key exists and not expired
	if expiry, ok := m.expiries[key]; ok && time.Now().After(expiry) {
		delete(m.data, key)
		delete(m.expiries, key)
	}

	if _, exists := m.data[key]; exists {
		return false, nil
	}

	m.data[key] = value
	if expiration > 0 {
		m.expiries[key] = time.Now().Add(expiration)
	}
	return true, nil
}

// Exists checks if keys exist
func (m *MockRedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var count int64
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			count++
		}
	}
	return count, nil
}

// Expire sets expiration on a key
func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if m.OnExpire != nil {
		return m.OnExpire(key, expiration)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.expiries[key] = time.Now().Add(expiration)
	return nil
}

// LPush prepends values to a list
func (m *MockRedisClient) LPush(ctx context.Context, key string, values ...interface{}) error {
	if m.OnLPush != nil {
		return m.OnLPush(key, values...)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.lists[key] == nil {
		m.lists[key] = []string{}
	}

	for _, v := range values {
		m.lists[key] = append([]string{v.(string)}, m.lists[key]...)
	}
	return nil
}

// BRPop pops from the end of a list with blocking
func (m *MockRedisClient) BRPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	if m.OnBRPop != nil {
		return m.OnBRPop(timeout, keys...)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		if list, ok := m.lists[key]; ok && len(list) > 0 {
			val := list[len(list)-1]
			m.lists[key] = list[:len(list)-1]
			return []string{key, val}, nil
		}
	}
	return nil, redis.Nil
}

// LLen returns the length of a list
func (m *MockRedisClient) LLen(ctx context.Context, key string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if list, ok := m.lists[key]; ok {
		return int64(len(list)), nil
	}
	return 0, nil
}

// LRange returns a range of list elements
func (m *MockRedisClient) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list, ok := m.lists[key]
	if !ok {
		return []string{}, nil
	}

	length := int64(len(list))
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start > stop {
		return []string{}, nil
	}

	return list[start : stop+1], nil
}

// LRem removes elements from a list
func (m *MockRedisClient) LRem(ctx context.Context, key string, count int64, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	list, ok := m.lists[key]
	if !ok {
		return nil
	}

	valStr := value.(string)
	newList := []string{}
	removed := int64(0)

	for _, v := range list {
		if v == valStr && (count == 0 || removed < count) {
			removed++
			continue
		}
		newList = append(newList, v)
	}

	m.lists[key] = newList
	return nil
}

// HSet sets hash fields
func (m *MockRedisClient) HSet(ctx context.Context, key string, values ...interface{}) error {
	if m.OnHSet != nil {
		return m.OnHSet(key, values...)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.hashes[key] == nil {
		m.hashes[key] = make(map[string]string)
	}

	for i := 0; i < len(values); i += 2 {
		field := values[i].(string)
		value := values[i+1].(string)
		m.hashes[key][field] = value
	}
	return nil
}

// HGet gets a hash field
func (m *MockRedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	if m.OnHGet != nil {
		return m.OnHGet(key, field)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	hash, ok := m.hashes[key]
	if !ok {
		return "", redis.Nil
	}

	val, ok := hash[field]
	if !ok {
		return "", redis.Nil
	}
	return val, nil
}

// HGetAll gets all hash fields
func (m *MockRedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if m.OnHGetAll != nil {
		return m.OnHGetAll(key)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	hash, ok := m.hashes[key]
	if !ok {
		return map[string]string{}, nil
	}

	result := make(map[string]string)
	for k, v := range hash {
		result[k] = v
	}
	return result, nil
}

// HDel deletes hash fields
func (m *MockRedisClient) HDel(ctx context.Context, key string, fields ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	hash, ok := m.hashes[key]
	if !ok {
		return nil
	}

	for _, field := range fields {
		delete(hash, field)
	}
	return nil
}

// HIncrBy increments a hash field
func (m *MockRedisClient) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.hashes[key] == nil {
		m.hashes[key] = make(map[string]string)
	}

	var val int64
	if v, ok := m.hashes[key][field]; ok {
		fmt.Sscanf(v, "%d", &val)
	}
	val += incr
	m.hashes[key][field] = fmt.Sprintf("%d", val)
	return val, nil
}

// SAdd adds members to a set
func (m *MockRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sets[key] == nil {
		m.sets[key] = make(map[string]struct{})
	}

	for _, member := range members {
		m.sets[key][member.(string)] = struct{}{}
	}
	return nil
}

// SRem removes members from a set
func (m *MockRedisClient) SRem(ctx context.Context, key string, members ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	set, ok := m.sets[key]
	if !ok {
		return nil
	}

	for _, member := range members {
		delete(set, member.(string))
	}
	return nil
}

// SMembers returns all set members
func (m *MockRedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	set, ok := m.sets[key]
	if !ok {
		return []string{}, nil
	}

	result := make([]string, 0, len(set))
	for member := range set {
		result = append(result, member)
	}
	return result, nil
}

// ZAdd adds members to a sorted set
func (m *MockRedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	if m.OnZAdd != nil {
		return m.OnZAdd(key, members...)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.zsets[key] == nil {
		m.zsets[key] = make(map[string]float64)
	}

	for _, z := range members {
		m.zsets[key][z.Member.(string)] = z.Score
	}
	return nil
}

// ZRem removes members from a sorted set
func (m *MockRedisClient) ZRem(ctx context.Context, key string, members ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	zset, ok := m.zsets[key]
	if !ok {
		return nil
	}

	for _, member := range members {
		delete(zset, member.(string))
	}
	return nil
}

// ZCard returns the cardinality of a sorted set
func (m *MockRedisClient) ZCard(ctx context.Context, key string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	zset, ok := m.zsets[key]
	if !ok {
		return 0, nil
	}
	return int64(len(zset)), nil
}

// ZRangeByScore returns members in a score range
func (m *MockRedisClient) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	if m.OnZRangeByScore != nil {
		return m.OnZRangeByScore(key, opt)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	zset, ok := m.zsets[key]
	if !ok {
		return []string{}, nil
	}

	var minScore, maxScore float64
	if opt.Min == "-inf" {
		minScore = -1e18
	} else {
		fmt.Sscanf(opt.Min, "%f", &minScore)
	}
	if opt.Max == "+inf" {
		maxScore = 1e18
	} else {
		fmt.Sscanf(opt.Max, "%f", &maxScore)
	}

	result := []string{}
	for member, score := range zset {
		if score >= minScore && score <= maxScore {
			result = append(result, member)
		}
	}
	return result, nil
}

// Ping pings the server
func (m *MockRedisClient) Ping(ctx context.Context) error {
	return nil
}

// FlushDB flushes the database
func (m *MockRedisClient) FlushDB(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]string)
	m.expiries = make(map[string]time.Time)
	m.lists = make(map[string][]string)
	m.sets = make(map[string]map[string]struct{})
	m.hashes = make(map[string]map[string]string)
	m.zsets = make(map[string]map[string]float64)
	return nil
}

// Close closes the client
func (m *MockRedisClient) Close() error {
	return nil
}
