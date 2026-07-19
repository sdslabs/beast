FROM python:3.12-alpine

WORKDIR /app

COPY requirements.txt /app/

RUN pip install --no-cache-dir --requirement requirements.txt

RUN addgroup -S docs && adduser -S -G docs docs

COPY ./docs/ /app/docs/
COPY mkdocs.yml /app/
RUN chown -R docs:docs /app
EXPOSE 8000

USER docs

CMD ["mkdocs", "serve", "-a", "0.0.0.0:8000"]
