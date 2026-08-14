FROM golang:1.25-alpine AS build
WORKDIR /src
COPY automation/ ./
RUN go mod download && go build -o /out/automation .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/automation /app/automation
EXPOSE 8091
CMD ["/app/automation"]
