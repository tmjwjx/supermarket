# 只编译 inventory 配置在启动时挂到 /configs/app.yaml
FROM golang:1.26.5 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/inventory ./app/inventory/cmd/inventory

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
COPY --from=build /out/inventory /usr/local/bin/inventory
EXPOSE 8002 9002
ENTRYPOINT ["inventory"]
CMD ["-conf", "/configs/app.yaml"]
