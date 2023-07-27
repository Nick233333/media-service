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
# COPY service/media/api/etc/media-api.yaml /app/etc/media-api.yaml
COPY service/media/api /app/service/media/api
WORKDIR /app/service/media/api
RUN go build -ldflags="-s -w" -o media 

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai
ENV TZ Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/service/media/api/media /app/media
COPY --from=builder /app/service/media/api/etc/media-api.yaml /app/etc/media-api.yaml

CMD ["./media", "-f", "etc/media-api.yaml"]
