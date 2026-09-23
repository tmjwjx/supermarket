# 只编译 user 配置在启动时挂到 /configs/app.yaml
FROM golang:1.26.5 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/user ./app/user/cmd/user

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
COPY --from=build /out/user /usr/local/bin/user
EXPOSE 8000 9000
ENTRYPOINT ["user"]
CMD ["-conf", "/configs/app.yaml"]
