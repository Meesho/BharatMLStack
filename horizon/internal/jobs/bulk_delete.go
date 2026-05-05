package jobs

import (
	"fmt"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/jobs/bulkdeletestrategy"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/schedule"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

type Task interface {
	ScheduledTask()
}

var (
	scheduledOnce sync.Once
	task          Task
)

type TaskImpl struct {
	scheduleJobRepo             schedule.ScheduleJobRepository
	serviceDeployableConfigRepo servicedeployableconfig.ServiceDeployableRepository
	strategySelectorImpl        bulkdeletestrategy.StrategySelectorImpl
}

func InitScheduledTask(config configs.Configs) Task {
	scheduledOnce.Do(func() {
		connection, err := infra.SQL.GetConnection()
		if err != nil {
			log.Panic().Err(err).Msg("Failed to get SQL connection")
		}
		sqlConn, ok := connection.(*infra.SQLConnection)
		if !ok {
			log.Panic().Msg("Failed to cast connection to SQLConnection")
		}
		repo, err := schedule.NewRepository(sqlConn)
		if err != nil {
			fmt.Println("Failed to create schedule job repository:", err)
			return
		}
		serviceDeployableConfigRepo, err := servicedeployableconfig.NewRepository(sqlConn)
		if err != nil {
			fmt.Println("Failed to create service deployable config repository:", err)
			return
		}
		task = &TaskImpl{
			scheduleJobRepo:             repo,
			serviceDeployableConfigRepo: serviceDeployableConfigRepo,
			strategySelectorImpl:        bulkdeletestrategy.Init(config),
		}
	})

	return task
}

func (t *TaskImpl) ScheduledTask() {
	today := time.Now()
	err := t.scheduleJobRepo.Create(&schedule.ScheduleJob{
		EntryDate: time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC),
	})

	if err != nil {
		log.Info().Msg("Already schedule in any other pod: " + err.Error())
		return
	}

	log.Info().Msg("Running Bulk Delete Scheduled Task from job at " + time.Now().Format(time.RFC1123))

	serviceDeployableList, err := t.serviceDeployableConfigRepo.GetAllActive()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get all active service deployable")
		return
	}

	for _, serviceDeployable := range serviceDeployableList {
		strategy, err := t.strategySelectorImpl.GetBulkDeleteStrategy(serviceDeployable.Service)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get bulk delete strategy")
			continue
		}
		err = strategy.ProcessBulkDelete(serviceDeployable)
		if err != nil {
			log.Error().Err(err).Msg(fmt.Sprintf("Failed to process bulk delete for service deployable: %d", serviceDeployable.ID))
		}
	}
}
