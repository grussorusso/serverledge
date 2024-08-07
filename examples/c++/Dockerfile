FROM grussorusso/serverledge-base AS BASE
FROM alpine:3.17.0

RUN apk update && \
    apk add --no-cache \
    build-base

WORKDIR /app

COPY rapidjson/ ./rapidjson/
COPY function.cpp .
COPY Makefile .

RUN make

WORKDIR /
# Required: install the executor as /executor
COPY --from=BASE /executor /executor
CMD /executor

ENV CUSTOM_CMD "/app/function"
