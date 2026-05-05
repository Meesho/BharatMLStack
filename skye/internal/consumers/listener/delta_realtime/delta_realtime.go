package delta_realtime

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/handler/indexer"
	"github.com/Meesho/BharatMLStack/skye/internal/consumers/listener/realtime"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

var (
	rtConsumer Consumer
	rtOnce     sync.Once
)

type RealTimeDeltaConsumer struct {
	configManager        config.Manager
	qdrantIndexerHandler indexer.Handler
	rateLimiters         map[int]*rate.Limiter
	partitionChans       []chan partitionJob
}

type partitionJob struct {
	partition    int
	event        indexer.Event
	commitOffset kafka.TopicPartition
	seekOffset   kafka.TopicPartition
	commit       func()
	seek         func()
}

func newRealTimeDeltaConsumer() Consumer {
	if rtConsumer == nil {
		rtOnce.Do(func() {
			configManager := config.NewManager(config.DefaultVersion)
			qdrantIndexerHandler := indexer.NewHandler(indexer.QDRANT)

			consumer := &RealTimeDeltaConsumer{
				configManager:        configManager,
				qdrantIndexerHandler: qdrantIndexerHandler,
				rateLimiters:         make(map[int]*rate.Limiter),
				partitionChans:       make([]chan partitionJob, 256),
			}

			// initialize rate limiters
			configuredRateLimiters := configManager.GetRateLimiters()
			for key, value := range configuredRateLimiters {
				if value.RateLimit == 0 {
					value.RateLimit = 1
				}
				if value.BurstLimit == 0 {
					value.BurstLimit = 1
				}
				consumer.rateLimiters[key] = rate.NewLimiter(rate.Limit(value.RateLimit), value.BurstLimit)
			}

			// start one worker per partition (0..255)
			for p := 0; p < 256; p++ {
				ch := make(chan partitionJob, 500)
				consumer.partitionChans[p] = ch
				go func(partition int, ch chan partitionJob, consumer *RealTimeDeltaConsumer) {
					defer func() {
						if r := recover(); r != nil {
							log.Error().Msgf("Recovered from panic in delta realtime worker for partition %d: %v", partition, r)
						}
						ct := 0
						for job := range ch {
							ct++
							job.seek()
							if ct > 0 {
								break
							}
						}
						close(ch)
						consumer.partitionChans[partition] = make(chan partitionJob, 500)
						ProcessPartitionJob(consumer, partition, ch)
					}()
					ProcessPartitionJob(consumer, partition, ch)
				}(p, ch, consumer)
			}

			rtConsumer = consumer
		})
	}
	return rtConsumer
}

func ProcessPartitionJob(r *RealTimeDeltaConsumer, partition int, ch chan partitionJob) {
	for job := range ch {
		allItems := []struct {
			eventType indexer.EventType
			eventData indexer.Data
		}{}
		model, variant, entity := "", "", ""
		for eventType, event := range job.event.Data {
			for _, eventData := range event {
				model = eventData.Model
				variant = eventData.Variant
				entity = eventData.Entity
				variantConfig, err := r.configManager.GetVariantConfig(eventData.Entity, eventData.Model, eventData.Variant)
				if err != nil {
					log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", eventData.Entity, eventData.Model, eventData.Variant, err)
				}
				if eventData.Version != variantConfig.VectorDbWriteVersion && eventData.Version != variantConfig.VectorDbReadVersion {
					log.Error().Msgf("Version mismatch for entity %s, model %s, variant %s: %v", eventData.Entity, eventData.Model, eventData.Variant, eventData.Version)
					metric.Count("version_error", 1, []string{"entity", eventData.Entity, "model", eventData.Model, "variant", eventData.Variant})
					continue
				}
				allItems = append(allItems, struct {
					eventType indexer.EventType
					eventData indexer.Data
				}{eventType, eventData})
			}
		}

		variantConfig, err := r.configManager.GetVariantConfig(entity, model, variant)
		if err != nil {
			log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", entity, model, variant, err)
		}
		if variantConfig.RateLimiter.RateLimit == 0 {
			metric.Gauge("partition_rt_consumption_stopped", 1, []string{"partition", strconv.Itoa(partition), "entity", entity, "model", model, "variant", variant})
			time.Sleep(time.Second * 5)
			job.seek()
			continue
		}

		// Determine chunk size using limiter burst when available
		limiter, hasLimiter := r.rateLimiters[partition]
		chunkSize := 0
		if hasLimiter && limiter != nil && limiter.Burst() > 0 {
			chunkSize = limiter.Burst()
		} else {
			chunkSize = len(allItems)
		}
		failed := false
		chunksProcessed := 0
		for i := 0; i < len(allItems); i += chunkSize {
			n := chunkSize
			if rem := len(allItems) - i; rem < n {
				n = rem
			}

			// Rate limit this chunk if limiter is configured
			if hasLimiter && limiter != nil {
				metric.Gauge("partition_rt_consumption_stopped", 0, []string{"partition", strconv.Itoa(partition), "entity", allItems[0].eventData.Entity, "model", allItems[0].eventData.Model, "variant", allItems[0].eventData.Variant})
				if err := limiter.WaitN(context.Background(), n); err != nil {
					log.Error().Msgf("Rate limiter Error for partition %v: %v", partition, err)
					job.seek()
					failed = true
					break
				}
			}
			microEvent := indexer.Event{Data: make(map[indexer.EventType][]indexer.Data)}
			for j := i; j < i+n; j++ {
				item := allItems[j]
				microEvent.Data[item.eventType] = append(microEvent.Data[item.eventType], item.eventData)
			}
			if err := r.qdrantIndexerHandler.Process(microEvent); err != nil {
				log.Error().Msgf("Error processing delta event for partition %v: %v", partition, err)
				job.seek()
				failed = true
				break
			}

			metric.Count("qdrant_events", int64(n), []string{"partition", strconv.Itoa(partition), "entity", allItems[0].eventData.Entity, "model", allItems[0].eventData.Model, "variant", allItems[0].eventData.Variant})
			chunksProcessed++
		}
		if chunksProcessed > 0 && len(allItems) > 0 {
			averageBatchSize := len(allItems) / chunksProcessed
			metric.Gauge("qdrant_avg_batch_size", float64(averageBatchSize), []string{"partition", strconv.Itoa(partition), "entity", allItems[0].eventData.Entity, "model", allItems[0].eventData.Model, "variant", allItems[0].eventData.Variant})
		}
		if failed {
			continue
		}
		job.commit()
	}
}

func (r *RealTimeDeltaConsumer) Process(events []realtime.DeltaEvent, c *kafka.Consumer) error {
	if err := r.ProcessDeltaEvent(events, c); err != nil {
		metric.Count("realtime_consumer_event_error", int64(len(events)), []string{})
		log.Error().Msg("Error processing combined indexer event")
		return err
	}

	return nil
}

func (r *RealTimeDeltaConsumer) ProcessDeltaEvent(events []realtime.DeltaEvent, c *kafka.Consumer) error {
	EventsMap := make(map[int]indexer.Event)
	PartitionToCommitOffsetMap := make(map[int]kafka.TopicPartition)
	PartitionToSeekOffsetMap := make(map[int]kafka.TopicPartition)
	for _, event := range events {
		if event.EventType == "UPSERT" && len(event.Vectors) == 0 {
			continue
		}
		variantConfig, err := r.configManager.GetVariantConfig(event.Entity, event.Model, event.Variant)
		if err != nil {
			log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", event.Entity, event.Model, event.Variant, err)
			return err
		}
		if _, ok := EventsMap[variantConfig.RTPartition]; !ok {
			EventsMap[variantConfig.RTPartition] = indexer.Event{
				Data: make(map[indexer.EventType][]indexer.Data),
			}
		}
		if _, ok := PartitionToCommitOffsetMap[variantConfig.RTPartition]; !ok {
			PartitionToCommitOffsetMap[variantConfig.RTPartition] = event.TopicPartition
		} else {
			currentOffset := PartitionToCommitOffsetMap[variantConfig.RTPartition]
			if event.TopicPartition.Offset > currentOffset.Offset {
				PartitionToCommitOffsetMap[variantConfig.RTPartition] = event.TopicPartition
			}
		}
		if _, ok := PartitionToSeekOffsetMap[variantConfig.RTPartition]; !ok {
			PartitionToSeekOffsetMap[variantConfig.RTPartition] = event.TopicPartition
		} else {
			currentOffset := PartitionToSeekOffsetMap[variantConfig.RTPartition]
			if event.TopicPartition.Offset < currentOffset.Offset {
				PartitionToSeekOffsetMap[variantConfig.RTPartition] = event.TopicPartition
			}
		}
		EventsMap[variantConfig.RTPartition].Data[indexer.EventType(event.EventType)] = append(EventsMap[variantConfig.RTPartition].Data[indexer.EventType(event.EventType)], indexer.Data{
			Entity:  event.Entity,
			Model:   event.Model,
			Variant: event.Variant,
			Version: event.Version,
			Id:      event.Id,
			Payload: event.Payload,
			Vectors: event.Vectors,
		})
	}
	for partition, event := range EventsMap {
		// Enqueue job for sequential processing to the fixed partition channel
		ch := r.partitionChans[partition]
		ch <- partitionJob{
			partition:    partition,
			event:        event,
			commitOffset: PartitionToCommitOffsetMap[partition],
			seekOffset:   PartitionToSeekOffsetMap[partition],
			commit:       func() { c.CommitOffsets([]kafka.TopicPartition{PartitionToCommitOffsetMap[partition]}) },
			seek:         func() { c.SeekPartitions([]kafka.TopicPartition{PartitionToSeekOffsetMap[partition]}) },
		}
	}

	return nil
}

func (r *RealTimeDeltaConsumer) RefreshRateLimiters(key, _, eventType string) error {
	log.Info().Msgf("Rate limiter change detected - Key: %s, EventType: %s", key, eventType)

	entity, model, variant := r.extractRateLimiterKey(key)
	if entity == "" || model == "" || variant == "" {
		// log.Error().Msgf("Could not extract rate limiter key from path: %s", key)
		return nil
	}
	variantConfig, err := r.configManager.GetVariantConfig(entity, model, variant)
	if err != nil {
		log.Error().Msgf("Error getting variant config for entity %s, model %s, variant %s: %v", entity, model, variant, err)
		return err
	}

	if eventType == "DELETE" {
		log.Info().Msgf("Removed rate limiter: %v", variantConfig.RTPartition)
		return nil
	}

	allRateLimiters := r.configManager.GetRateLimiters()
	if variantConfig.RTPartition == 0 {
		log.Error().Msgf("RTPartition is 0 for entity %s, model %s, variant %s", entity, model, variant)
		return nil
	}
	if variantConfig.RateLimiter.RateLimit == 0 {
		variantConfig.RateLimiter.RateLimit = 1
	}
	if variantConfig.RateLimiter.BurstLimit == 0 {
		variantConfig.RateLimiter.BurstLimit = 1
	}
	newConfig, exists := allRateLimiters[variantConfig.RTPartition]
	if !exists {
		log.Error().Msgf("Rate limiter key %v not found in config", variantConfig.RTPartition)
		return nil
	}

	rateLimiter := rate.NewLimiter(
		rate.Limit(newConfig.RateLimit),
		newConfig.BurstLimit,
	)
	r.rateLimiters[variantConfig.RTPartition] = rateLimiter

	log.Error().Msgf("Updated rate limiter %v: rate=%d, burst=%d",
		variantConfig.RTPartition, newConfig.RateLimit, newConfig.BurstLimit)

	return nil
}

func (r *RealTimeDeltaConsumer) extractRateLimiterKey(key string) (string, string, string) {
	parts := strings.Split(key, "/")
	for _, part := range parts {
		if part == "rate-limit" || part == "burst-limit" {
			return parts[4], parts[6], parts[8]
		}
	}
	return "", "", ""
}
