FROM python:3.8-slim

WORKDIR /app

RUN apt-get update && \
    apt-get install -y postgresql-client curl && \
    rm -rf /var/lib/apt/lists/*

COPY requirements.txt .

RUN pip install -r requirements.txt

COPY . .

RUN mkdir -p /app/app/static/js && \
    chmod -R 755 /app/app/static

RUN chmod +x entrypoint.sh

# Add a non-root user and switch to this user

# Change ownership of the app directory to the non-root user
RUN useradd -m hacker \
    && chown -R root:root /app \
    && chmod -R 755 /app \
    && chmod -R o-w /app

USER hacker

EXPOSE 80
ENTRYPOINT ["./entrypoint.sh"]