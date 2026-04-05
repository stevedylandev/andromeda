FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends sqlite3 awscli cron && \
    rm -rf /var/lib/apt/lists/*

COPY backup.sh /usr/local/bin/backup.sh
RUN chmod +x /usr/local/bin/backup.sh

COPY crontab /etc/cron.d/backup-cron
RUN chmod 0644 /etc/cron.d/backup-cron && crontab /etc/cron.d/backup-cron

# Pass environment variables to cron jobs
CMD printenv | grep -E '^(R2_|AWS_|RETENTION)' >> /etc/environment && cron -f
