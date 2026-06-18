FROM golang:1.25.10-alpine3.22 AS builder

ENV APP_PATH=/Users/durgaprasad/my/observeblity

WORKDIR ${APP_PATH}

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags "-s -w"  -o my_observatlity_image


# second stage

FROM scratch

COPY --from=builder /Users/durgaprasad/my/observeblity/my_observatlity_image /app/observe

CMD ["/app/my_observatlity_image","run"]

