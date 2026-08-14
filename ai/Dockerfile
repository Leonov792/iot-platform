FROM golang:1.25-alpine AS build
WORKDIR /src
COPY ai/ ./
RUN go mod download && go build -o /out/ai .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/ai /app/ai
EXPOSE 8095
CMD ["/app/ai"]
