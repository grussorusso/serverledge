FROM golang:1.17-alpine AS build

WORKDIR /sedge

COPY . ./
RUN go mod download


RUN go build -o /executor ./cmd/executor/executor.go

##
## Deploy
##
FROM python:3.10-alpine3.14

WORKDIR /

COPY --from=build /executor /
COPY images/python310/entrypoint.py /
RUN mkdir -p /app

CMD /executor
