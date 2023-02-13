FROM python:3.10-alpine3.14

WORKDIR /

COPY images/python310/executor.py /
RUN mkdir -p /app
RUN pip install requests
RUN pip install pandas
RUN pip install numpy
RUN pip install scikit-learn

CMD python3 /executor.py
