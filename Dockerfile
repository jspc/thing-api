FROM golang:1.18.0-alpine as build

RUN echo "app:x:1000:1000::/_nonesuch:/bin/sodall" > /tmp/mini.passwd

WORKDIR /app
RUN apk add --update upx

# Only add go specific sources to avoid large build contexts
ADD go.* ./
ADD *.go ./
ADD docs/ ./docs

RUN CGO_ENABLED=0 go build -o app -ldflags="-s -w" && upx app

FROM scratch

COPY --from=build /tmp/mini.passwd /etc/passwd
USER 1000

ENV GIN_MODE=release

CMD ["/app"]

COPY --from=build /app/app /app
