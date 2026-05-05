package server

import (
	"net/http"
	"strconv"

	"github.com/Meesho/BharatMLStack/skye/pkg/httpframework"
	"github.com/rs/zerolog/log"
)

func InitServer(port int) {
	if port == 0 {
		log.Panic().Msg("PORT not set")
	}

	err := http.ListenAndServe(":"+strconv.Itoa(port), httpframework.Instance())
	log.Info().Msg("Server Started")

	if err != nil {
		// panic and stop the app if server does not start
		log.Panic().Msgf("There's an error while starting the server!, error - %v", err)
	}

}
