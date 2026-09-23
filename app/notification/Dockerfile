# 只编译 notification 配置在启动时挂到 /configs/app.yaml
FROM golang:1.26.5 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/notification ./app/notification/cmd/notification

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
COPY --from=build /out/notification /usr/local/bin/notification
EXPOSE 8005 9005
ENTRYPOINT ["notification"]
CMD ["-conf", "/configs/app.yaml"]
