FROM golang:alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED 0
ENV GO111MODULE on
ENV GOPROXY https://goproxy.cn,direct
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk update --no-cache && apk add --no-cache tzdata

WORKDIR /app

ADD go.mod .
ADD go.sum .
RUN go mod download
ADD ./common ./common
COPY service/media/consumer /app/service/media/consumer
WORKDIR /app/service/media/consumer
RUN go build -ldflags="-s -w"

FROM alpine

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai
ENV TZ Asia/Shanghai
RUN set -ex \
    # change apk source repo
    && sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk update \
    && apk add ffmpeg 

WORKDIR /app
COPY --from=builder /app/service/media/consumer/asckey/ /app/asckey/
COPY --from=builder /app/service/media/consumer/consumer /app/consumer
COPY --from=builder /app/service/media/consumer/etc/consumer.yaml /app/etc/consumer.yaml

CMD ["./consumer", "-f", "etc/consumer.yaml"]