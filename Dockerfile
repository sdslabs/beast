# Use Python 2.7 as the base image
FROM python:2.7

WORKDIR /app

COPY requirements.txt /app/

RUN pip install --no-cache-dir -r requirements.txt

COPY ./docs/ /app/docs/
COPY mkdocs.yml /app/
EXPOSE 8000

CMD ["mkdocs", "serve", "-a", "0.0.0.0:8000"]
