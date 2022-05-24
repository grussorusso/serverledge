FROM grussorusso/serverledge-base as BASE

# Extend any image you want, e.g.;
FROM python:3.8.1

# Required: install the executor as /executor
COPY --from=BASE /executor /
CMD /executor

# Required: this is the command representing your function
ENV CUSTOM_CMD "python /function.py"

# Install your code and any dependency, e.g.:
COPY function.py /
