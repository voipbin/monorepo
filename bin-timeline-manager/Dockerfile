# build
FROM public.ecr.aws/docker/library/golang:1.25-alpine AS build

WORKDIR /app
COPY ./ .
RUN mkdir -p /app/bin
RUN cd bin-timeline-manager && go build -o /app/bin/ ./cmd/...

# run
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app/bin/
COPY --from=build /app/bin /app/bin
COPY --from=build /app/bin-timeline-manager/migrations /app/bin/migrations

CMD ["./timeline-manager"]
