package driveradminhttp

import (
	"errors"
	"net/http"

	cf_http_handlers "code.cloudfoundry.org/cfhttp/handlers"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/smbdriver/driveradmin"
	"code.cloudfoundry.org/voldriver/driverhttp"
	"github.com/tedsuo/rata"
)

func NewHandler(logger lager.Logger, client driveradmin.DriverAdmin) (http.Handler, error) {
	logger = logger.Session("server")
	logger.Info("start")
	defer logger.Info("end")

	var handlers = rata.Handlers{
		driveradmin.EvacuateRoute: newEvacuateHandler(logger, client),
		driveradmin.PingRoute:     newPingHandler(logger, client),
	}

	return rata.NewRouter(driveradmin.Routes, handlers)
}

func newEvacuateHandler(logger lager.Logger, client driveradmin.DriverAdmin) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logger := logger.Session("handle-evacuate")
		logger.Info("start")
		defer logger.Info("end")

		env := driverhttp.EnvWithMonitor(logger, req.Context(), w)

		response := client.Evacuate(env)
		if response.Err != "" {
			logger.Error("failed-evacuating", errors.New(response.Err))
			cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, response)
			return
		}

		cf_http_handlers.WriteJSONResponse(w, http.StatusOK, response)
	}
}

func newPingHandler(logger lager.Logger, client driveradmin.DriverAdmin) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		logger := logger.Session("handle-ping")
		logger.Info("start")
		defer logger.Info("end")

		env := driverhttp.EnvWithMonitor(logger, req.Context(), w)

		response := client.Ping(env)
		if response.Err != "" {
			logger.Error("failed-pinging", errors.New(response.Err))
			cf_http_handlers.WriteJSONResponse(w, http.StatusInternalServerError, response)
			return
		}

		cf_http_handlers.WriteJSONResponse(w, http.StatusOK, response)
	}
}
