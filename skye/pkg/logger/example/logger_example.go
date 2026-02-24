package loggerexample

import (
	"github.com/Meesho/BharatMLStack/skye/pkg/logger"
	"github.com/rs/zerolog/log"
)

//lint:ignore U1000 example entrypoint: go run .
func main() {

	logger.InitLogger("test-app", "DEBUG")
	log.Info().Msgf("This is an error message")

}
