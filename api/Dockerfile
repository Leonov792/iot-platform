FROM golang:1.25-alpine AS build
WORKDIR /src
COPY api/ ./
RUN go mod download && go build -o /out/api ./cmd/api

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/api /app/api
EXPOSE 8080
CMD ["/app/api"]
