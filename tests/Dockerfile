FROM python:3-alpine
WORKDIR /code
RUN pip install pytest
RUN pip install requests
COPY integration_tests.py .
ENTRYPOINT ["pytest","integration_tests.py"]