FROM golang:1.25-alpine AS build
WORKDIR /src
COPY eco/ ./
RUN go mod download && go build -o /out/eco .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/eco /app/eco
EXPOSE 8096
CMD ["/app/eco"]
