FROM node:17-alpine

RUN mkdir -p /app
COPY bin/executor /

COPY images/nodejs17/entrypoint.js /

CMD /executor
