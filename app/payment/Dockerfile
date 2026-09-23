# 只编译 payment 配置在启动时挂到 /configs/app.yaml
FROM golang:1.26.5 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/payment ./app/payment/cmd/payment

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
COPY --from=build /out/payment /usr/local/bin/payment
EXPOSE 8004 9004
ENTRYPOINT ["payment"]
CMD ["-conf", "/configs/app.yaml"]
