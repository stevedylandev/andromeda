FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends sqlite3 awscli cron && \
    rm -rf /var/lib/apt/lists/*

COPY apps/backup/backup.sh /usr/local/bin/backup.sh
RUN chmod +x /usr/local/bin/backup.sh

# restore.sh ships for reference; run it on the HOST (it shells out to `docker run`
# and the backup container mounts the data volumes read-only).
COPY apps/backup/restore.sh /usr/local/bin/restore.sh
RUN chmod +x /usr/local/bin/restore.sh

COPY apps/backup/crontab /etc/cron.d/backup-cron
RUN chmod 0644 /etc/cron.d/backup-cron && crontab /etc/cron.d/backup-cron

CMD ["sh", "-c", "printenv | grep -E '^(R2_|AWS_|RETENTION)' >> /etc/environment && cron -f"]
