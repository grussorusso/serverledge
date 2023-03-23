FROM python:3.10-buster

WORKDIR /

COPY images/custom-python/executor.py /
RUN mkdir -p /app

RUN apt update
RUN pip install requests
RUN pip install pandas
RUN pip install numpy
RUN pip install scikit-learn

CMD python3 /executor.py