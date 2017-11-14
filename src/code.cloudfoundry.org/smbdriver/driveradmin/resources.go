package driveradmin

import (
	"code.cloudfoundry.org/voldriver"
	"github.com/tedsuo/rata"
)

const (
	EvacuateRoute = "evacuate"
	PingRoute     = "ping"
)

var Routes = rata.Routes{
	{Path: "/evacuate", Method: "GET", Name: EvacuateRoute},
	{Path: "/ping", Method: "GET", Name: PingRoute},
}

//go:generate counterfeiter -o smbdriverfakes/fake_driver_admin.go src/code.cloudfoundry.org/smbdriver/driveradmin DriverAdmin
type DriverAdmin interface {
	Evacuate(env voldriver.Env) ErrorResponse
	Ping(env voldriver.Env) ErrorResponse
}

type ErrorResponse struct {
	Err string
}

//go:generate counterfeiter -o smbdriverfakes/fake_drainable.go src/code.cloudfoundry.org/smbdriver/driveradmin Drainable
type Drainable interface {
	Drain(env voldriver.Env) error
}
