package scheduler

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/jobs"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

var initSchedulerOnce sync.Once

func Init(config configs.Configs) {
	initSchedulerOnce.Do(func() {
		c := cron.New(cron.WithSeconds())

		cronExpression := config.ScheduledCronExpression
		if cronExpression == "" {
			log.Warn().Msg("SCHEDULED_CRON_EXPRESSION not found in the config, skipping scheduler")
			return
		}

		task := jobs.InitScheduledTask(config)

		if task == nil {
			log.Fatal().Msg("Failed to initialize scheduled task")
			return
		}

		_, err := c.AddFunc(cronExpression, task.ScheduledTask)
		if err != nil {
			log.Error().Err(err).Msg("Failed to schedule the task")
			return
		}

		c.Start()
	})
}
