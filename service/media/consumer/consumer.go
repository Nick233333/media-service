package main

import (
	"flag"

	"curriculum/service/media/consumer/internal/config"
	"curriculum/service/media/consumer/internal/service"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

var configFile = flag.String("f", "etc/consumer.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	srv := service.NewService(c)
	queue := kq.MustNewQueue(c.Kafka, kq.WithHandle(srv.Consume))
	defer queue.Stop()
	logx.DisableStat()

	queue.Start()
}
