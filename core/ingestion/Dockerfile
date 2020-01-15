FROM python:3.6
RUN pip install pipenv
WORKDIR /src
COPY Pipfile* ./
RUN pipenv install --dev --system --skip-lock
ADD . .

RUN black --check .
RUN isort -c

RUN pip install ./scan
RUN pip install ./upload
RUN pip install ./blobio

RUN pytest upload scan

ENTRYPOINT [ "pytest", "tests" ]
