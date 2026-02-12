FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
COPY *.go ./
COPY lua/ ./lua/
RUN go mod tidy -e && CGO_ENABLED=0 go build -o /factorio-exporter .

FROM alpine:3.21
COPY --from=build /factorio-exporter /factorio-exporter
COPY lua/ /lua/
ENTRYPOINT ["/factorio-exporter"]
