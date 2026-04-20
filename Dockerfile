FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/app .

FROM scratch
COPY --from=build /out/app /app
EXPOSE 8080
ENTRYPOINT ["/app"]
