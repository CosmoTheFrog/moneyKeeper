FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
RUN CGO_ENABLED=0 go build -o /out/moneykeeper ./cmd/moneykeeper

FROM alpine:3.22
COPY --from=build /out/moneykeeper /usr/local/bin/moneykeeper
ENTRYPOINT ["moneykeeper"]
