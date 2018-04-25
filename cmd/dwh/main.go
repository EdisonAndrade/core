package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	log "github.com/noxiouz/zapctx/ctxlog"
	"github.com/sonm-io/core/cmd"
	"github.com/sonm-io/core/insonmnia/dwh"
	"github.com/sonm-io/core/insonmnia/logging"
	"github.com/sonm-io/core/util"
	"go.uber.org/zap"
)

var (
	configFlag  string
	versionFlag bool
	appVersion  string
)

func main() {
	cmd.NewCmd("dwh", appVersion, &configFlag, &versionFlag, run).Execute()
}

func run() {
	cfg, err := dwh.NewConfig(configFlag)
	if err != nil {
		log.G(context.Background()).Error("failed to load config", zap.Error(err))
		os.Exit(1)
	}

	logger := logging.BuildLogger(cfg.Logging.Level)
	ctx := log.WithLogger(context.Background(), logger)

	key, err := cfg.Eth.LoadKey()
	if err != nil {
		log.G(ctx).Error("failed load private key", zap.Error(err))
		os.Exit(1)
	}

	w, err := dwh.NewDWH(ctx, cfg, key)
	if err != nil {
		log.G(ctx).Error("cannot create new DWH", zap.Error(err))
		os.Exit(1)
	}

	go util.StartPrometheus(ctx, cfg.MetricsListenAddr)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		w.Stop()
	}()

	log.G(ctx).Info("starting DWH service", zap.String("grpc_addr", cfg.GRPCListenAddr),
		zap.String("http_addr", cfg.HTTPListenAddr))
	if err := w.Serve(); err != nil {
		log.G(ctx).Error("DWH stopped", zap.Error(err))
		os.Exit(1)
	}
}