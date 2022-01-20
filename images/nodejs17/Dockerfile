FROM golang:1.17-alpine AS build

WORKDIR /sedge

COPY . ./
RUN go mod download

RUN go build -o /executor ./cmd/executor/executor.go

##
## Deploy
##
FROM node:17-alpine

WORKDIR /

COPY --from=build /executor /
COPY images/nodejs17/entrypoint.js /
RUN mkdir -p /app

CMD /executor
