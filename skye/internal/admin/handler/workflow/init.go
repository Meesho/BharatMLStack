package workflow

import (
	"sync"

	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
)

var (
	machine        StateMachine
	once           sync.Once
	DefaultVersion = 1
	appConfig      structs.Configs
	initOnce       sync.Once
)

func Init() {
	initOnce.Do(func() {
		appConfig = structs.GetAppConfig().Configs
	})
}

func NewStateMachine(version int) StateMachine {
	switch version {
	case DefaultVersion:
		return initModelStateMachine()
	default:
		return nil
	}
}

func SetInstance(provider StateMachine) {
	machine = provider
	once.Do(func() {})
}
