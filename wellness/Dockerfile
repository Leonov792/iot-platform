FROM golang:1.25-alpine AS build
WORKDIR /src
COPY wellness/ ./
RUN go mod download && go build -o /out/wellness .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/wellness /app/wellness
EXPOSE 8097
CMD ["/app/wellness"]
