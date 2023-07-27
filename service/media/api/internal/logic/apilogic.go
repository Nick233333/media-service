package logic

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"curriculum/common/errorx"
	"curriculum/service/media/api/internal/svc"
	"curriculum/service/media/api/internal/types"

	"github.com/Shopify/sarama"
	"github.com/zeromicro/go-zero/core/logx"
)

type ApiLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 2接口逻辑
func NewApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApiLogic {
	return &ApiLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 1验证
func (l *ApiLogic) Api(req *types.TransformReq) (resp *types.TransformReply, err error) {

	if req.NotifyUrl == "" {
		return nil, errorx.NewDefaultError("notify_url 不能为空!")
	}

	fileExtension := filepath.Ext(req.MediaUrl)

	mediaTypes := l.svcCtx.Config.MediaTypes
	checkMediaTypeResult := true
	// 循环验证转换标准参数
	for _, t := range mediaTypes {

		if fileExtension == t {
			checkMediaTypeResult = false
		}
	}
	// 验证后缀
	if checkMediaTypeResult {
		return nil, errorx.NewDefaultError("media_url 参数后缀错误")
	}

	if req.Standard == "" {
		return nil, errorx.NewDefaultError("standard 不能为空!")
	}

	//视频转换标准
	standards := l.svcCtx.Config.Standards

	checkStandardResult := true
	// 循环验证转换标准参数
	for _, s := range standards {
		if req.Standard == s {
			checkStandardResult = false
		}
	}

	//验证不通过
	if checkStandardResult {
		return nil, errorx.NewDefaultError("standard 参数错误")
	}

	// 定义多个域名前缀
	prefixes := l.svcCtx.Config.Domains

	checkSuffixResult := true
	// 循环验证域名参数
	for _, prefix := range prefixes {
		if strings.HasPrefix(req.MediaUrl, prefix) {
			checkSuffixResult = false
		}
	}

	//不匹配域名
	if checkSuffixResult {
		return nil, errorx.NewDefaultError("media_url 参数域名错误")
	}

	data, err := json.Marshal(req)

	if err != nil {
		logx.Error("json.Marshal failed:", err)
		return
	}

	// 创建配置
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll          // 发送完数据需要leader和follower都确认
	config.Producer.Partitioner = sarama.NewRandomPartitioner //写到随机分区中，我们默认设置32个分区
	config.Producer.Return.Successes = true                   // 成功交付的消息将在success channel返回

	// 创建生产者
	producer, err := sarama.NewSyncProducer(l.svcCtx.Config.Kafka.Addrs, config)
	if err != nil {
		logx.Error("Failed to create producer:", err)
		return
	}
	defer producer.Close()

	// 构建消息
	message := &sarama.ProducerMessage{
		Topic: l.svcCtx.Config.Kafka.MediaTopic,
		Value: sarama.StringEncoder(data),
	}

	// 发送消息到任意分区
	_, _, err = producer.SendMessage(message)
	if err != nil {
		logx.Info("Failed to send message:", err)
	} else {
		logx.Info("Message sent successfully")
	}

	return &types.TransformReply{
		Code: 200,
		Msg:  "success",
		Data: "",
	}, nil
}
