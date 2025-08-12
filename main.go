package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var version = "dev"

type Config struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	OutputFile    string
	Workers       int
	BatchSize     int
	LogLevel      string
}

type RedisEntry struct {
	Key   string      `json:"key"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
	TTL   int64       `json:"ttl,omitempty"`
}

type Exporter struct {
	client *redis.Client
	config Config
}

func NewExporter(config Config) *Exporter {
	rdb := redis.NewClient(&redis.Options{
		Addr:         config.RedisAddr,
		Password:     config.RedisPassword,
		DB:           config.RedisDB,
		PoolSize:     config.Workers * 2, // More connections for higher concurrency
		MinIdleConns: config.Workers,     // Keep connections warm
		PoolTimeout:  30 * time.Second,   // Longer pool timeout
		ReadTimeout:  10 * time.Second,   // Longer read timeout for large values
		WriteTimeout: 10 * time.Second,   // Longer write timeout
	})

	return &Exporter{
		client: rdb,
		config: config,
	}
}

func (e *Exporter) getValueByType(ctx context.Context, key string, keyType string) (interface{}, error) {
	switch keyType {
	case "string":
		return e.client.Get(ctx, key).Result()
	case "list":
		return e.client.LRange(ctx, key, 0, -1).Result()
	case "set":
		return e.client.SMembers(ctx, key).Result()
	case "zset":
		return e.client.ZRangeWithScores(ctx, key, 0, -1).Result()
	case "hash":
		return e.client.HGetAll(ctx, key).Result()
	case "stream":
		return e.client.XRange(ctx, key, "-", "+").Result()
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}
}

func (e *Exporter) processKey(ctx context.Context, key string) (*RedisEntry, error) {
	keyType, err := e.client.Type(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get type for key %s: %w", key, err)
	}

	value, err := e.getValueByType(ctx, key, keyType)
	if err != nil {
		return nil, fmt.Errorf("failed to get value for key %s: %w", key, err)
	}

	ttl, err := e.client.TTL(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get TTL for key %s: %w", key, err)
	}

	entry := &RedisEntry{
		Key:   key,
		Type:  keyType,
		Value: value,
	}

	if ttl > 0 {
		entry.TTL = int64(ttl.Seconds())
	}

	return entry, nil
}

func (e *Exporter) worker(ctx context.Context, keysChan <-chan string, resultsChan chan<- *RedisEntry, wg *sync.WaitGroup) {
	defer wg.Done()

	for key := range keysChan {
		select {
		case <-ctx.Done():
			return
		default:
			entry, err := e.processKey(ctx, key)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"key": key,
				}).Error("Error processing key: ", err)
				continue
			}
			resultsChan <- entry
		}
	}
}

func (e *Exporter) Export(ctx context.Context) error {
	logrus.WithFields(logrus.Fields{
		"output_file": e.config.OutputFile,
		"workers":     e.config.Workers,
		"batch_size":  e.config.BatchSize,
	}).Info("Starting Redis export")

	file, err := os.Create(e.config.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	keysChan := make(chan string, e.config.BatchSize)
	resultsChan := make(chan *RedisEntry, e.config.BatchSize)

	var wg sync.WaitGroup
	for i := 0; i < e.config.Workers; i++ {
		wg.Add(1)
		go e.worker(ctx, keysChan, resultsChan, &wg)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	encoder := json.NewEncoder(file)
	file.WriteString("[\n")

	var processed int64
	var firstEntry = true

	go func() {
		defer close(keysChan)

		iter := e.client.Scan(ctx, 0, "*", int64(e.config.BatchSize)).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			select {
			case keysChan <- key:
			case <-ctx.Done():
				return
			}
		}

		if err := iter.Err(); err != nil {
			logrus.Error("Error during key scanning: ", err)
		}
	}()

	startTime := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case entry, ok := <-resultsChan:
			if !ok {
				file.WriteString("\n]")
				elapsed := time.Since(startTime)
				rate := float64(processed) / elapsed.Seconds()
				logrus.WithFields(logrus.Fields{
					"total_keys":       processed,
					"total_duration":   elapsed.Round(time.Second),
					"avg_keys_per_sec": rate,
				}).Info("Export completed successfully")
				return nil
			}

			if !firstEntry {
				file.WriteString(",\n")
			} else {
				firstEntry = false
			}

			if err := encoder.Encode(entry); err != nil {
				logrus.WithFields(logrus.Fields{
					"key": entry.Key,
				}).Error("Error encoding entry: ", err)
				continue
			}

			processed++

		case <-ticker.C:
			elapsed := time.Since(startTime)
			rate := float64(processed) / elapsed.Seconds()
			logrus.WithFields(logrus.Fields{
				"processed_keys": processed,
				"keys_per_sec":   rate,
				"elapsed":        elapsed.Round(time.Second),
			}).Info("Export progress")

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

var config Config

var rootCmd = &cobra.Command{
	Use:     "redis-export",
	Short:   "High-performance Redis database exporter to JSON",
	Long:    "Export all keys and values from a Redis database to JSON format with concurrent processing",
	Version: version,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cmd.Flags().Changed("addr") && !cmd.Flags().Changed("output") {
			return cmd.Help()
		}

		// Configure logrus
		level, err := logrus.ParseLevel(config.LogLevel)
		if err != nil {
			return fmt.Errorf("invalid log level: %w", err)
		}
		logrus.SetLevel(level)
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})

		exporter := NewExporter(config)
		defer exporter.client.Close()

		ctx := context.Background()

		logrus.WithField("redis_addr", config.RedisAddr).Info("Connecting to Redis")
		pong, err := exporter.client.Ping(ctx).Result()
		if err != nil {
			return fmt.Errorf("failed to connect to Redis: %w", err)
		}
		logrus.WithField("response", pong).Info("Successfully connected to Redis")

		return exporter.Export(ctx)
	},
}

func init() {
	rootCmd.Flags().StringVarP(&config.RedisAddr, "addr", "a", "localhost:6379", "Redis server address")
	rootCmd.Flags().StringVarP(&config.RedisPassword, "password", "p", "", "Redis password")
	rootCmd.Flags().IntVarP(&config.RedisDB, "db", "d", 0, "Redis database number")
	rootCmd.Flags().StringVarP(&config.OutputFile, "output", "o", "redis_export.json", "Output JSON file")
	rootCmd.Flags().IntVarP(&config.Workers, "workers", "w", runtime.NumCPU()*2, "Number of worker goroutines")
	rootCmd.Flags().IntVarP(&config.BatchSize, "batch", "b", 1000, "Batch size for key scanning")
	rootCmd.Flags().StringVarP(&config.LogLevel, "log-level", "l", "info", "Log level (trace, debug, info, warn, error, fatal, panic)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
