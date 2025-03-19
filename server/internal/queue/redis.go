package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/araaavind/zoko-im/internal/data"
	"github.com/araaavind/zoko-im/internal/websocket"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	StreamKey        string
	ConsumerGroup    string
	ConsumerName     string
	BlockingDuration time.Duration
	MaxRetries       int
	RetryDelay       time.Duration
	BatchSize        int
	dlq              struct {
		key string
	}
}

type MessageQueue struct {
	client *redis.Client
	config Config
	logger *slog.Logger
	models data.Models
	hub    *websocket.Hub
}

func NewMessageQueue(client *redis.Client, config Config, logger *slog.Logger, models data.Models, hub *websocket.Hub) *MessageQueue {
	return &MessageQueue{
		client: client,
		config: config,
		logger: logger,
		models: models,
		hub:    hub,
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
		"batch size", q.config.BatchSize,
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
				Count:    int64(q.config.BatchSize),
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
				err = q.models.Messages.BulkInsert(ctx, messages)
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

			// Acknowledge messages or send to DLQ
			if success {
				q.logger.Info("message batch processed successfully",
					"count", len(messages))
				if len(messageIDs) > 0 {
					q.client.XAck(ctx, q.config.StreamKey, q.config.ConsumerGroup, messageIDs...)
				}
			} else {
				q.logger.Error("message batch processing failed after retries",
					"batch_size", len(messages))
				// Send to DLQ
				for _, msg := range messages {
					msgJSON, _ := json.Marshal(msg)
					q.client.XAdd(ctx, &redis.XAddArgs{
						Stream: q.config.dlq.key,
						Values: map[string]any{
							"message": string(msgJSON),
						},
					})
				}
				// Still acknowledge to prevent blocking
				if len(messageIDs) > 0 {
					q.client.XAck(ctx, q.config.StreamKey, q.config.ConsumerGroup, messageIDs...)
				}
			}
		}
	}
}

func (q *MessageQueue) ProcessDLQ(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read a batch of messages from the DLQ
			streams, err := q.client.XRead(ctx, &redis.XReadArgs{
				Streams: []string{q.config.dlq.key, "$"},
				Block:   q.config.BlockingDuration,
				Count:   100,
			}).Result()

			if err != nil && err != redis.Nil {
				q.logger.Error("Error reading from DLQ", "error", err)
				time.Sleep(q.config.RetryDelay)
				continue
			}

			if len(streams) == 0 || len(streams[0].Messages) == 0 {
				continue
			}

			// Process messages from DLQ
			for _, redisMsg := range streams[0].Messages {
				messageJSON, ok := redisMsg.Values["message"].(string)
				if !ok {
					q.logger.Error("invalid message format in DLQ", "message_id", redisMsg.ID)
					continue
				}

				var msg data.Message
				err := json.Unmarshal([]byte(messageJSON), &msg)
				if err != nil {
					q.logger.Error("failed to unmarshal message from DLQ", "error", err, "message_id", redisMsg.ID)
					continue
				}

				// Attempt to reprocess the message
				err = q.models.Messages.Insert(ctx, &msg)
				if err != nil {
					q.logger.Error("failed to reprocess message from DLQ", "error", err, "message_id", redisMsg.ID)
					continue
				}

				// Acknowledge the message from DLQ
				q.logger.Info("message reprocessed successfully from DLQ")
				q.client.XAck(ctx, q.config.dlq.key, "dlq_processor", redisMsg.ID)
			}
		}
	}
}
