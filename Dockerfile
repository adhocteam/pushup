FROM golang:1.19.1-alpine3.15
RUN apk update upgrade
RUN apk add build-base git
WORKDIR /usr/src/app
# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
COPY . .
RUN make
ENTRYPOINT [ "/go/bin/pushup" ]