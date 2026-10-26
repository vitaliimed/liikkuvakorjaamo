FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test ./... && CGO_ENABLED=0 go build -buildvcs=false -trimpath -o /out/server .
FROM alpine:3.23
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/server .
COPY *.html ./
COPY static ./static
COPY templates ./templates
USER app
ENV PORT=8080
CMD ["/app/server"]
