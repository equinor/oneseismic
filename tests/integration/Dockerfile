FROM python:3.6-alpine
RUN apk add --no-cache curl pkgconfig openssl-dev libffi-dev musl-dev make gcc
WORKDIR /code
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY integration_tests.py .
ENTRYPOINT ["pytest","-v","integration_tests.py"]
