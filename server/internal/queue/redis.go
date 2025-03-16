package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/araaavind/zoko-im/internal/data"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	StreamKey        string
	ConsumerGroup    string
	ConsumerName     string
	BlockingDuration time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
}

type MessageQueue struct {
	client *redis.Client
	config Config
	logger *slog.Logger
	models data.Models
}

func NewMessageQueue(client *redis.Client, config Config, logger *slog.Logger, models data.Models) *MessageQueue {
	return &MessageQueue{
		client: client,
		config: config,
		logger: logger,
		models: models,
	}
}

// EnqueueMessage adds a message to the Redis stream
func (q *MessageQueue) EnqueueMessage(ctx context.Context, message *data.Message) error {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.config.StreamKey,
		Values: map[string]any{
			"message": string(messageJSON),
		},
	}).Err()
}

// ProcessMessages starts a worker to process messages from the stream
func (q *MessageQueue) ProcessMessages(ctx context.Context) error {
	// Create consumer group if it doesn't exist
	err := q.client.XGroupCreateMkStream(ctx, q.config.StreamKey, q.config.ConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}

	batchSize := 100

	q.logger.Info(
		"message worker started",
		"group", q.config.ConsumerGroup,
		"consumer", q.config.ConsumerName,
		"batch size", batchSize,
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read a batch of messages
			streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    q.config.ConsumerGroup,
				Consumer: q.config.ConsumerName,
				Streams:  []string{q.config.StreamKey, ">"},
				Block:    q.config.BlockingDuration,
				Count:    int64(batchSize),
			}).Result()

			if err == redis.Nil {
				continue
			} else if err != nil {
				q.logger.Error("Error reading from consumer group", "error", err)
				time.Sleep(q.config.RetryDelay)
				continue
			}

			if len(streams) == 0 || len(streams[0].Messages) == 0 {
				continue
			}

			messages := make([]*data.Message, 0, len(streams[0].Messages))
			messageIDs := make([]string, 0, len(streams[0].Messages))

			// Parse messages
			for _, redisMsg := range streams[0].Messages {
				messageJSON, ok := redisMsg.Values["message"].(string)
				if !ok {
					q.logger.Error("invalid message format", "message_id", redisMsg.ID)
					q.client.XAck(ctx, q.config.StreamKey, q.config.ConsumerGroup, redisMsg.ID)
					continue
				}

				var msg data.Message
				err := json.Unmarshal([]byte(messageJSON), &msg)
				if err != nil {
					q.logger.Error("failed to unmarshal message", "error", err, "message_id", redisMsg.ID)
					q.client.XAck(ctx, q.config.StreamKey, q.config.ConsumerGroup, redisMsg.ID)
					continue
				}

				messages = append(messages, &msg)
				messageIDs = append(messageIDs, redisMsg.ID)
			}

			// Process batch with retries
			success := false
			for i := range q.config.MaxRetries {
				err = q.models.Messages.BulkInsert(messages)
				if err == nil {
					success = true
					break
				}
				q.logger.Error("failed to process message batch",
					"error", err,
					"retry", i+1,
					"batch_size", len(messages))
				time.Sleep(q.config.RetryDelay)
			}

			// Acknowledge messages
			if success {
				q.logger.Info("message batch processed successfully",
					"count", len(messages))
				// Acknowledge all messages in the batch
				if len(messageIDs) > 0 {
					q.client.XAck(ctx, q.config.StreamKey, q.config.ConsumerGroup, messageIDs...)
				}
			} else {
				q.logger.Error("message batch processing failed after retries",
					"batch_size", len(messages))
				// Still acknowledge to prevent blocking
				if len(messageIDs) > 0 {
					q.client.XAck(ctx, q.config.StreamKey, q.config.ConsumerGroup, messageIDs...)
				}
			}
		}
	}
}
