package env

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/actuator"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/bootstrap"
	"go.uber.org/fx"
)

var Module = &bootstrap.Module{
	Name: "actuator-env",
	Precedence: actuator.MinActuatorPrecedence,
	Options: []fx.Option{
		fx.Provide(BindEnvProperties),
	},
}

func init() {
	bootstrap.Register(Module)
}

type regDI struct {
	fx.In
	Registrar     *actuator.Registrar
	MgtProperties actuator.ManagementProperties
	AppContext    *bootstrap.ApplicationContext
	Properties    EnvProperties
}

func Register(di regDI) {
	ep := new(di)
	di.Registrar.Register(ep)
}