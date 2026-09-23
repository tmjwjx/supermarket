# 只编译 admin 配置在启动时挂到 /configs/app.yaml
FROM golang:1.26.5 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/admin ./app/admin/cmd/admin

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
COPY --from=build /out/admin /usr/local/bin/admin
EXPOSE 8006 9006
ENTRYPOINT ["admin"]
CMD ["-conf", "/configs/app.yaml"]
