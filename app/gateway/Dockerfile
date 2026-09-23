# 只编译 gateway 配置在启动时挂到 /configs/app.yaml
FROM golang:1.26.5 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/gateway ./app/gateway/cmd/gateway

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
COPY --from=build /out/gateway /usr/local/bin/gateway
EXPOSE 8080
ENTRYPOINT ["gateway"]
CMD ["-conf", "/configs/app.yaml"]
