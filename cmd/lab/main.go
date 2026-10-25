package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"hw2-mongo/internal/config"
	"hw2-mongo/internal/lab"
	catalogService "hw2-mongo/internal/service/catalog"
	datasetService "hw2-mongo/internal/service/dataset"
	failoverService "hw2-mongo/internal/service/failover"
	ordersService "hw2-mongo/internal/service/orders"
	purchaseService "hw2-mongo/internal/service/purchase"
	reportService "hw2-mongo/internal/service/report"
	connManager "hw2-mongo/internal/storage/mongo/conn-manager"
	ordersStorage "hw2-mongo/internal/storage/mongo/orders"
	productsStorage "hw2-mongo/internal/storage/mongo/products"
	txmanager "hw2-mongo/internal/storage/mongo/tx-manager"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Parse()
	if err != nil {
		panic(err)
	}

	logger := config.InitLogger(cfg.Logger)

	command := "help"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	if command == "help" || command == "--help" || command == "-h" {
		lab.PrintHelp()

		return
	}

	if command == "fixtures" {
		lab.PrintFixtures()

		return
	}

	names, ok := lab.RunScenarioNames(command)
	if !ok {
		logger.Error().Str("command", command).Msg("неизвестная команда; см. help")
		os.Exit(1)
	}

	resetMode := lab.ResetModeData
	if len(os.Args) > 2 && os.Args[2] == "index" {
		resetMode = lab.ResetModeIndex
	}

	connMng, err := connManager.New(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not create connection manager")
	}
	defer connMng.Close()

	txManager := txmanager.New(connMng)
	productsSt := productsStorage.New(connMng)
	ordersSt := ordersStorage.New(connMng)

	env := &lab.Env{
		Client:   connMng.GetClient(),
		Catalog:  catalogService.New(logger, productsSt),
		Orders:   ordersService.New(logger, ordersSt),
		Report:   reportService.New(logger, ordersSt),
		Purchase: purchaseService.New(logger, productsSt, ordersSt, txManager),
		Failover: failoverService.New(logger, connMng),
		Dataset: datasetService.New(logger, connMng.GetClient(),
			cfg.Mongo.DBName, productsSt, ordersSt),
		Products:      productsSt,
		OrdersSt:      ordersSt,
		ServerVersion: cfg.Mongo.ServerVersion,
		ResetMode:     resetMode,
	}

	runner := lab.NewRunner(env, cfg.ResultsPath)

	logger.Info().Str("command", command).Msg("running scenario")

	if err := runner.Run(ctx, names); err != nil {
		logger.Fatal().Err(err).Msg("scenario failed")
	}
}
