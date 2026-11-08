FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test ./... && CGO_ENABLED=0 go build -buildvcs=false -trimpath -o /out/crm ./cmd/crm
FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/crm .
USER app
ENV PORT=8080
CMD ["/app/crm"]
