FROM golang:1.25-alpine AS build
WORKDIR /src
COPY alerts/ ./
RUN go mod download && go build -o /out/alerts .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/alerts /app/alerts
CMD ["/app/alerts"]
