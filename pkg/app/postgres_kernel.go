package app

import (
	"context"
	"github.com/exgamer/gosdk-core/pkg/app"
	"github.com/exgamer/gosdk-core/pkg/di"
	database "github.com/exgamer/gosdk-postgres-core/pkg/helpers"
)

const DbKernelName = "postgres"

type PostgresKernel struct {
	reg *PostgresGormRegistry
}

func (m *PostgresKernel) Name() string {
	return DbKernelName
}

func (m *PostgresKernel) Init(a *app.App) error {
	dbConfig, err := database.InitPostgresDbConfig()

	if err != nil {
		return err
	}

	reg := NewPostgresGormRegistry(database.InitPostgresGormConnection)
	reg.AddDefaultConnection(dbConfig)
	di.Register(a.Container, reg)
	m.reg = reg

	return nil
}

func (m *PostgresKernel) Start(a *app.App) error {

	return nil
}

func (m *PostgresKernel) Stop(ctx context.Context) error {
	if m.reg == nil {
		return nil
	}

	return m.reg.CloseAll()
}
