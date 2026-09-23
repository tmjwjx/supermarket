# 只编译 order 配置在启动时挂到 /configs/app.yaml
FROM golang:1.26.5 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/order ./app/order/cmd/order

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
COPY --from=build /out/order /usr/local/bin/order
EXPOSE 8003 9003
ENTRYPOINT ["order"]
CMD ["-conf", "/configs/app.yaml"]
