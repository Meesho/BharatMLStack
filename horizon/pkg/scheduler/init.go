package scheduler

import (
	"log"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/jobs"
	"github.com/robfig/cron/v3"
)

var initSchedulerOnce sync.Once

func Init(config configs.Configs) {
	initSchedulerOnce.Do(func() {
		c := cron.New(cron.WithSeconds())

		cronExpression := config.ScheduledCronExpression
		if cronExpression == "" {
			log.Fatal("SCHEDULED_CRON_EXPRESSION not found in the config")
		}

		task := jobs.InitScheduledTask(config)

		if task == nil {
			log.Fatal("Failed to initialize scheduled task")
			return
		}

		_, err := c.AddFunc(cronExpression, task.ScheduledTask)
		if err != nil {
			log.Fatalf("Failed to schedule the task: %v", err)
		}

		c.Start()
	})
}
