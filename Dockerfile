FROM golang:1.19 AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags "-s -w -extldflags '-static'" -o /myproject ./build/cmd/myproject
FROM alpine
COPY --from=builder /myproject /
EXPOSE 8080
ENTRYPOINT [ "/myproject" ]
