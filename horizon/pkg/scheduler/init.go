package scheduler

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/jobs"
	skyeJobs "github.com/Meesho/BharatMLStack/horizon/internal/skye/jobs"
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

func InitVariantOnboardingScheduler(config configs.Configs) {
	c := cron.New(cron.WithSeconds())

	_, err := c.AddFunc(config.VariantOnboardingCronExpression, func() {
		job := skyeJobs.InitVariantOnboardingJob(config)
		job.Run()
	})

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to schedule variant onboarding job")
		return
	}

	c.Start()
	log.Info().Msg("Variant onboarding scheduler started (runs every 5 minutes)")
}

func InitVariantScaleUpScheduler(config configs.Configs) {
	c := cron.New(cron.WithSeconds())

	_, err := c.AddFunc(config.VariantScaleUpCronExpression, func() {
		job := skyeJobs.InitVariantScaleUpJob(config)
		job.Run()
	})

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to schedule variant scale up job")
		return
	}

	c.Start()
	log.Info().Msg("Variant Scale Up scheduler started (runs every 5 minutes)")
}
