package bootstrap

import (
	"context"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"os"
	"sort"
	"strings"
	"sync"
)

var once sync.Once
var bootstrapperInstance *Bootstrapper
var (
	initialContextOptions = []ContextOption{}
	startContextOptions   = []ContextOption{}
	stopContextOptions    = []ContextOption{}
)

type ContextOption func(ctx context.Context) context.Context

type Bootstrapper struct {
	modules []*Module
}

type App struct {
	*fx.App
	startCtxOptions []ContextOption
	stopCtxOptions  []ContextOption
}

// singleton pattern
func bootstrapper() *Bootstrapper {
	once.Do(func() {
		bootstrapperInstance = &Bootstrapper{
			modules: []*Module{applicationMainModule(), anonymousModule()},
		}
	})
	return bootstrapperInstance
}

func Register(m *Module) {
	b := bootstrapper()
	b.modules = append(b.modules, m)
}

func AddOptions(options...fx.Option) {
	m := anonymousModule()
	m.PriorityOptions = append(m.PriorityOptions, options...)
}

func AddInitialAppContextOptions(options...ContextOption) {
	initialContextOptions = append(initialContextOptions, options...)
}

func AddStartContextOptions(options...ContextOption) {
	startContextOptions = append(startContextOptions, options...)
}

func AddStopContextOptions(options...ContextOption) {
	stopContextOptions = append(stopContextOptions, options...)
}

func newApp(cmd *cobra.Command, priorityOptions []fx.Option, regularOptions []fx.Option) *App {
	DefaultModule.PriorityOptions = append(DefaultModule.PriorityOptions, fx.Supply(cmd))
	for _,o := range priorityOptions {
		applicationMainModule().PriorityOptions = append(applicationMainModule().PriorityOptions, o)
	}

	for _,o := range regularOptions {
		applicationMainModule().Options = append(applicationMainModule().Options, o)
	}

	b := bootstrapper()
	sort.SliceStable(b.modules, func(i, j int) bool { return b.modules[i].Precedence < b.modules[j].Precedence })

	// add priority options first
	var options []fx.Option
	for _,m := range b.modules {
		options = append(options, m.PriorityOptions...)
	}

	// add other options later
	for _,m := range b.modules {
		options = append(options, m.Options...)
	}

	// update application context before creating the app
	ctx := applicationContext.Context
	for _, opt := range initialContextOptions {
		ctx = opt(ctx)
	}
	applicationContext = applicationContext.withContext(ctx)

	// create App, which will kick off all fx options
	return &App{
		App: fx.New(options...),
		startCtxOptions: startContextOptions,
		stopCtxOptions: stopContextOptions,
	}
}

func (app *App) Run() {
	// to be revised:
	//  1. (Solved)	Support Timeout in bootstrap.Context and make cancellable context as startParent (swap startParent and child)
	//  2. (Solved) Restore logging
	done := app.Done()
	startParent, cancel := context.WithTimeout(applicationContext.Context, app.StartTimeout())
	for _, opt := range app.startCtxOptions {
		startParent = opt(startParent)
	}
	// This is so that we know that the context in the life cycle hook is the bootstrap context
	startCtx := applicationContext.withContext(startParent)
	defer cancel()

	if err := app.Start(startCtx); err != nil {
		logger.WithContext(startCtx).Errorf("Failed to start up: %v", err)
		exit(1)
	}

	// this line blocks until application shutting down
	printSignal(<-done)

	// shutdown sequence
	stopParent, cancel := context.WithTimeout(applicationContext.Context, app.StopTimeout())
	for _, opt := range app.stopCtxOptions {
		stopParent = opt(stopParent)
	}
	stopCtx := applicationContext.withContext(stopParent)
	defer cancel()

	if err := app.Stop(stopCtx); err != nil {
		logger.WithContext(stopCtx).Errorf("Failed to gracefully shutdown: %v", err)
		exit(1)
	}
}


func printSignal(signal os.Signal) {
	logger.Infof(strings.ToUpper(signal.String()))
}

func exit(code int) {
	os.Exit(code)
}
