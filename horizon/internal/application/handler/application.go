package handler

import (
	"fmt"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/application"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var (
	emptyResponse = ""
	activeTrue    = true
)

type applicationConfig struct {
	ApplicationRepo application.ApplicationRepository
}

func InitV1ConfigHandler() Config {
	if config == nil {
		conn, err := infra.SQL.GetConnection()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get SQL connection")
		}
		sqlConn := conn.(*infra.SQLConnection)

		ApplicationRepo, err := application.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create config repository")
		}

		config = &applicationConfig{
			ApplicationRepo: ApplicationRepo,
		}
	}
	return config
}

func (c *applicationConfig) Onboard(request OnboardRequest) (Response, error) {
	appToken := uuid.New().String()

	application := application.Application{
		AppToken:  appToken,
		Bu:        request.Payload.Bu,
		Team:      request.Payload.Team,
		Service:   request.Payload.Service,
		Active:    activeTrue,
		CreatedBy: request.CreatedBy,
	}

	err := c.ApplicationRepo.Create(&application)
	if err != nil {
		return Response{
			Error: err.Error(),
			Data:  Message{Message: emptyResponse},
		}, err
	}
	return Response{
		Error: emptyResponse,
		Data:  Message{Message: fmt.Sprintf("Application onboarded successfully with token: %s", appToken)},
	}, nil
}

func (c *applicationConfig) GetAll() (GetAllResponse, error) {
	applications, err := c.ApplicationRepo.GetAll()
	if err != nil {
		return GetAllResponse{}, err
	}

	response := []ApplicationConfig{}

	for _, application := range applications {
		response = append(response, ApplicationConfig{
			AppToken:  application.AppToken,
			Active:    application.Active,
			Bu:        application.Bu,
			Team:      application.Team,
			Service:   application.Service,
			CreatedBy: application.CreatedBy,
			UpdatedBy: application.UpdatedBy,
			CreatedAt: application.CreatedAt.Format(time.RFC3339),
			UpdatedAt: application.UpdatedAt.Format(time.RFC3339),
		})
	}
	return GetAllResponse{
		Data: response,
	}, nil
}

func (c *applicationConfig) Edit(request EditRequest) (Response, error) {
	application := application.Application{
		AppToken:  request.Payload.AppToken,
		Bu:        request.Payload.Bu,
		Team:      request.Payload.Team,
		Service:   request.Payload.Service,
		UpdatedBy: request.CreatedBy,
	}

	err := c.ApplicationRepo.Update(&application)
	if err != nil {
		return Response{
			Error: err.Error(),
			Data:  Message{Message: emptyResponse},
		}, err
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: fmt.Sprintf("Application updated successfully with token: %s", request.Payload.AppToken)},
	}, nil
}
