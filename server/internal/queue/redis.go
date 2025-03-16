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

	q.logger.Info(
		"message worker started",
		"group", q.config.ConsumerGroup,
		"consumer", q.config.ConsumerName,
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    q.config.ConsumerGroup,
				Consumer: q.config.ConsumerName,
				Streams:  []string{q.config.StreamKey, ">"},
				Block:    q.config.BlockingDuration,
				Count:    1,
			}).Result()

			if err == redis.Nil {
				continue
			} else if err != nil {
				q.logger.Error("Error reading from consumer group", "error", err)
				time.Sleep(q.config.RetryDelay)
				continue
			}

			for _, message := range streams[0].Messages {
				messageJSON, ok := message.Values["message"].(string)
				if !ok {
					q.logger.Error("invalid message format", "message_id", message.ID)
					q.client.XAck(ctx, q.config.StreamKey, q.config.ConsumerGroup, message.ID)
					continue
				}

				var msg data.Message
				err := json.Unmarshal([]byte(messageJSON), &msg)
				if err != nil {
					q.logger.Error("failed to unmarshal message", "error", err)
					q.client.XAck(ctx, q.config.StreamKey, q.config.ConsumerGroup, message.ID)
					continue
				}

				// Process message with retries
				success := false
				for i := range q.config.MaxRetries {
					err = q.models.Messages.Insert(&msg)
					if err == nil {
						success = true
						break
					}
					q.logger.Error("failed to process message",
						"error", err,
						"retry", i+1,
						"message_id", message.ID)
					time.Sleep(q.config.RetryDelay)
				}

				if success {
					q.logger.Info("message processed successfully",
						"message_id", message.ID)
					q.client.XAck(ctx, q.config.StreamKey, q.config.ConsumerGroup, message.ID)
				} else {
					// Move to dead letter queue or handle failure
					q.logger.Error("message processing failed after retries",
						"message_id", message.ID)
					// Still acknowledge to prevent blocking
					q.client.XAck(ctx, q.config.StreamKey, q.config.ConsumerGroup, message.ID)
				}
			}
		}
	}
}
