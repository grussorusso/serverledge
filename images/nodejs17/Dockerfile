FROM node:17-alpine

RUN mkdir -p /app
COPY executor /

COPY entrypoint.js /

CMD /executor
