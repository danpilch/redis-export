package main

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExporter_GetValueByType_String(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer func() { _ = db.Close() }()

	exporter := &Exporter{
		client: db,
		config: Config{},
	}

	ctx := context.Background()
	key := "test:string"
	expectedValue := "hello world"

	mock.ExpectGet(key).SetVal(expectedValue)

	value, err := exporter.getValueByType(ctx, key, "string")
	require.NoError(t, err)
	assert.Equal(t, expectedValue, value)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExporter_GetValueByType_List(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer func() { _ = db.Close() }()

	exporter := &Exporter{
		client: db,
		config: Config{},
	}

	ctx := context.Background()
	key := "test:list"
	expectedValue := []string{"item1", "item2", "item3"}

	mock.ExpectLRange(key, 0, -1).SetVal(expectedValue)

	value, err := exporter.getValueByType(ctx, key, "list")
	require.NoError(t, err)
	assert.Equal(t, expectedValue, value)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExporter_GetValueByType_Set(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer func() { _ = db.Close() }()

	exporter := &Exporter{
		client: db,
		config: Config{},
	}

	ctx := context.Background()
	key := "test:set"
	expectedValue := []string{"member1", "member2", "member3"}

	mock.ExpectSMembers(key).SetVal(expectedValue)

	value, err := exporter.getValueByType(ctx, key, "set")
	require.NoError(t, err)
	assert.Equal(t, expectedValue, value)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExporter_GetValueByType_Hash(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer func() { _ = db.Close() }()

	exporter := &Exporter{
		client: db,
		config: Config{},
	}

	ctx := context.Background()
	key := "test:hash"
	expectedValue := map[string]string{
		"field1": "value1",
		"field2": "value2",
	}

	mock.ExpectHGetAll(key).SetVal(expectedValue)

	value, err := exporter.getValueByType(ctx, key, "hash")
	require.NoError(t, err)
	assert.Equal(t, expectedValue, value)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExporter_GetValueByType_ZSet(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer func() { _ = db.Close() }()

	exporter := &Exporter{
		client: db,
		config: Config{},
	}

	ctx := context.Background()
	key := "test:zset"
	expectedValue := []redis.Z{
		{Score: 1.0, Member: "member1"},
		{Score: 2.0, Member: "member2"},
	}

	mock.ExpectZRangeWithScores(key, 0, -1).SetVal(expectedValue)

	value, err := exporter.getValueByType(ctx, key, "zset")
	require.NoError(t, err)
	assert.Equal(t, expectedValue, value)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExporter_ProcessKey_WithTTL(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer func() { _ = db.Close() }()

	exporter := &Exporter{
		client: db,
		config: Config{},
	}

	ctx := context.Background()
	key := "test:key"
	expectedType := "string"
	expectedValue := "test value"
	expectedTTL := 3600 * time.Second

	mock.ExpectType(key).SetVal(expectedType)
	mock.ExpectGet(key).SetVal(expectedValue)
	mock.ExpectTTL(key).SetVal(expectedTTL)

	entry, err := exporter.processKey(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, key, entry.Key)
	assert.Equal(t, expectedType, entry.Type)
	assert.Equal(t, expectedValue, entry.Value)
	assert.Equal(t, int64(3600), entry.TTL)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExporter_ProcessKey_NoTTL(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer func() { _ = db.Close() }()

	exporter := &Exporter{
		client: db,
		config: Config{},
	}

	ctx := context.Background()
	key := "test:key"
	expectedType := "string"
	expectedValue := "test value"

	mock.ExpectType(key).SetVal(expectedType)
	mock.ExpectGet(key).SetVal(expectedValue)
	mock.ExpectTTL(key).SetVal(-1 * time.Second)

	entry, err := exporter.processKey(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, key, entry.Key)
	assert.Equal(t, expectedType, entry.Type)
	assert.Equal(t, expectedValue, entry.Value)
	assert.Equal(t, int64(0), entry.TTL)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExporter_Worker(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer func() { _ = db.Close() }()

	exporter := &Exporter{
		client: db,
		config: Config{},
	}

	ctx := context.Background()
	keysChan := make(chan string, 2)
	resultsChan := make(chan *RedisEntry, 2)
	var wg sync.WaitGroup

	keys := []string{"key1", "key2"}
	for _, key := range keys {
		mock.ExpectType(key).SetVal("string")
		mock.ExpectGet(key).SetVal("value")
		mock.ExpectTTL(key).SetVal(-1 * time.Second)
	}

	keysChan <- "key1"
	keysChan <- "key2"
	close(keysChan)

	wg.Add(1)
	go exporter.worker(ctx, keysChan, resultsChan, &wg)

	wg.Wait()
	close(resultsChan)

	results := make([]*RedisEntry, 0)
	for entry := range resultsChan {
		results = append(results, entry)
	}

	assert.Len(t, results, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExporter_Worker_ContextCanceled(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer func() { _ = db.Close() }()

	exporter := &Exporter{
		client: db,
		config: Config{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	keysChan := make(chan string, 1)
	resultsChan := make(chan *RedisEntry, 1)
	var wg sync.WaitGroup

	keysChan <- "key1"
	cancel()

	wg.Add(1)
	go exporter.worker(ctx, keysChan, resultsChan, &wg)

	wg.Wait()
	close(resultsChan)

	results := make([]*RedisEntry, 0)
	for entry := range resultsChan {
		results = append(results, entry)
	}

	assert.Len(t, results, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExporter_Export_MockRedis(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer func() { _ = db.Close() }()

	config := Config{
		OutputFile: "test_mock_export.json",
		Workers:    1,
		BatchSize:  10,
	}

	exporter := &Exporter{
		client: db,
		config: config,
	}

	defer func() { _ = os.Remove(config.OutputFile) }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mock.ExpectScan(0, "*", int64(10)).SetVal([]string{"key1"}, 0)
	mock.ExpectType("key1").SetVal("string")
	mock.ExpectGet("key1").SetVal("value1")
	mock.ExpectTTL("key1").SetVal(-1 * time.Second)

	err := exporter.Export(ctx)
	require.NoError(t, err)

	_, err = os.Stat(config.OutputFile)
	assert.NoError(t, err, "Output file should be created")

	content, err := os.ReadFile(config.OutputFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "key1")
	assert.Contains(t, string(content), "value1")

	assert.NoError(t, mock.ExpectationsWereMet())
}
