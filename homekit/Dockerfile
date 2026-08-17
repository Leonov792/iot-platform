FROM golang:1.25-alpine AS build
WORKDIR /src
COPY homekit/ ./
RUN go mod download && go build -o /out/homekit .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/homekit /app/homekit
EXPOSE 51826
CMD ["/app/homekit"]
