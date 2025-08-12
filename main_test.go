package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisEntry_JSON(t *testing.T) {
	entry := RedisEntry{
		Key:   "test:key",
		Type:  "string",
		Value: "test value",
		TTL:   3600,
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var decoded RedisEntry
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, entry.Key, decoded.Key)
	assert.Equal(t, entry.Type, decoded.Type)
	assert.Equal(t, entry.Value, decoded.Value)
	assert.Equal(t, entry.TTL, decoded.TTL)
}

func TestRedisEntry_JSON_NoTTL(t *testing.T) {
	entry := RedisEntry{
		Key:   "test:key",
		Type:  "string",
		Value: "test value",
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var decoded RedisEntry
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, entry.Key, decoded.Key)
	assert.Equal(t, entry.Type, decoded.Type)
	assert.Equal(t, entry.Value, decoded.Value)
	assert.Equal(t, int64(0), decoded.TTL)
}

func TestGetValueByType_UnsupportedType(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer db.Close()

	exporter := &Exporter{
		client: db,
		config: Config{},
	}

	ctx := context.Background()
	_, err := exporter.getValueByType(ctx, "test", "unsupported")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported key type")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConfig_Defaults(t *testing.T) {
	config := Config{}
	assert.Empty(t, config.RedisAddr)
	assert.Empty(t, config.RedisPassword)
	assert.Equal(t, 0, config.RedisDB)
	assert.Empty(t, config.OutputFile)
	assert.Equal(t, 0, config.Workers)
	assert.Equal(t, 0, config.BatchSize)
}

func TestNewExporter(t *testing.T) {
	config := Config{
		RedisAddr:     "localhost:6379",
		RedisPassword: "secret",
		RedisDB:       1,
		OutputFile:    "test.json",
		Workers:       4,
		BatchSize:     100,
	}

	exporter := NewExporter(config)
	defer exporter.client.Close()

	assert.NotNil(t, exporter.client)
	assert.Equal(t, config, exporter.config)
}

func TestExporter_Export_FileCreation(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer db.Close()

	config := Config{
		OutputFile: "test_export_file.json",
		Workers:    1,
		BatchSize:  10,
	}

	exporter := &Exporter{
		client: db,
		config: config,
	}

	defer os.Remove(config.OutputFile)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	mock.ExpectScan(0, "*", int64(10)).SetVal([]string{"test:key"}, 0)
	mock.ExpectType("test:key").SetVal("string")
	mock.ExpectGet("test:key").SetVal("test value")
	mock.ExpectTTL("test:key").SetVal(-1 * time.Second)

	err := exporter.Export(ctx)
	require.NoError(t, err)

	_, err = os.Stat(config.OutputFile)
	assert.NoError(t, err, "Output file should be created")

	content, err := os.ReadFile(config.OutputFile)
	assert.NoError(t, err)
	assert.True(t, len(content) > 0, "Output file should not be empty")
	assert.Contains(t, string(content), "[", "Output should be valid JSON array")
	assert.Contains(t, string(content), "]", "Output should be valid JSON array")
	assert.Contains(t, string(content), "test:key")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExporter_Export_InvalidOutputPath(t *testing.T) {
	db, mock := redismock.NewClientMock()
	defer db.Close()

	config := Config{
		OutputFile: "/invalid/path/test.json",
		Workers:    1,
		BatchSize:  10,
	}

	exporter := &Exporter{
		client: db,
		config: config,
	}

	ctx := context.Background()
	err := exporter.Export(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create output file")
	assert.NoError(t, mock.ExpectationsWereMet())
}
