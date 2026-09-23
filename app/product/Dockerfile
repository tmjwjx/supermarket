# 只编译 product 配置在启动时挂到 /configs/app.yaml
FROM golang:1.26.5 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/product ./app/product/cmd/product

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
COPY --from=build /out/product /usr/local/bin/product
EXPOSE 8001 9001
ENTRYPOINT ["product"]
CMD ["-conf", "/configs/app.yaml"]
