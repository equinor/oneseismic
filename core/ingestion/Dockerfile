FROM python:3.6
RUN pip install pipenv
WORKDIR /src
COPY Pipfile* ./
RUN pipenv install --dev --system
ADD . .

RUN pip install ./scan
RUN pip install ./upload

RUN pytest upload scan

ENTRYPOINT [ "pytest", "tests" ]
