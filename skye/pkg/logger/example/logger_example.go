package loggerexample

import (
	"github.com/Meesho/BharatMLStack/skye/pkg/logger"
	"github.com/rs/zerolog/log"
)

func main() {

	logger.InitLogger("test-app", "DEBUG")
	log.Info().Msgf("This is an error message")

}
