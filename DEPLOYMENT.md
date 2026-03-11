# mri Production Deployment Guide

This guide covers deploying [project] for automated, unattended operation.

## Prerequisites

1. Runtime installed (e.g. Python 3.12+, Node 20+)
2. Dependencies installed
3. Secrets configured in `secrets.env`
4. Any credentials or tokens generated (see README.md setup)

## Initial Setup

### 1. Clone and Install

```bash
cd /opt   # or your preferred location
git clone https://github.com/[username]/[project].git
cd [project]

# Python example
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

### 2. Configure Secrets

```bash
cp secrets.env.example secrets.env
nano secrets.env   # fill in your values
chmod 600 secrets.env
```

### 3. Test the Setup

```bash
# Run in dry-run mode first
python main.py run --dry-run

# If successful, try a real run
python main.py run --limit 5
```

## Scheduling with Cron

### Setup

```bash
crontab -e

# Example: run every 4 hours
0 */4 * * * cd /opt/[project] && .venv/bin/python main.py run >> /var/log/[project]/cron.log 2>&1
```

### Wrapper Script (recommended)

Create `scripts/run_production.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="/opt/[project]"
LOG_DIR="/var/log/[project]"
LOG_FILE="$LOG_DIR/[project]-$(date +%Y%m%d).log"

mkdir -p "$LOG_DIR"
source "$PROJECT_DIR/.venv/bin/activate"
cd "$PROJECT_DIR"

echo "=== Run: $(date) ===" >> "$LOG_FILE"
python main.py run >> "$LOG_FILE" 2>&1
echo "" >> "$LOG_FILE"
```

### Cron Schedule Examples

```bash
# Every 4 hours
0 */4 * * * /opt/[project]/scripts/run_production.sh

# Twice daily
0 8,18 * * * /opt/[project]/scripts/run_production.sh

# Once daily at midnight
0 0 * * * /opt/[project]/scripts/run_production.sh
```

## Scheduling with Systemd Timer (Linux)

### Service File

`/etc/systemd/system/[project].service`:

```ini
[Unit]
Description=mri
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
User=[project]
WorkingDirectory=/opt/[project]
Environment="PATH=/opt/[project]/.venv/bin:/usr/local/bin:/usr/bin"
ExecStart=/opt/[project]/.venv/bin/python /opt/[project]/main.py run

StandardOutput=journal
StandardError=journal
SyslogIdentifier=[project]

PrivateTmp=yes
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/[project]
```

### Timer File

`/etc/systemd/system/[project].timer`:

```ini
[Unit]
Description=mri Timer
Requires=[project].service

[Timer]
OnCalendar=*-*-* 00,04,08,12,16,20:00:00
OnBootSec=5min
RandomizedDelaySec=10min

[Install]
WantedBy=timers.target
```

### Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable [project].timer
sudo systemctl start [project].timer
sudo systemctl status [project].timer
```

## Security

### Dedicated User

```bash
sudo useradd -r -s /usr/sbin/nologin -d /opt/[project] [project]
sudo chown -R [project]:[project] /opt/[project]
sudo chmod 600 /opt/[project]/secrets.env
```

### File Permissions

```bash
chmod 600 secrets.env
chmod 755 scripts/*.sh
```

### Verify .gitignore

```bash
cat .gitignore | grep secrets
./scripts/security_scan.sh
```

## Monitoring and Logging

### View Logs

```bash
# Cron logs
tail -f /var/log/[project]/cron.log

# Systemd logs
sudo journalctl -u [project].service -f
sudo journalctl -u [project].service --since "2 hours ago"
sudo journalctl -u [project].service -p err
```

### Log Rotation

`/etc/logrotate.d/[project]`:

```
/var/log/[project]/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
}
```

## Troubleshooting

### Permission Errors

```bash
ls -la /opt/[project]/
sudo chown [project]:[project] /opt/[project]/[relevant file]
sudo chmod 600 /opt/[project]/secrets.env
```

### Service Won't Start

```bash
sudo systemctl status [project].service
sudo journalctl -u [project].service -n 50
```

### Manual Test Run

```bash
sudo -u [project] /opt/[project]/.venv/bin/python /opt/[project]/main.py run --dry-run
```

## Backup and Recovery

```bash
# Backup any state files (e.g. database, token)
cp [project].db [project].db.backup-$(date +%Y%m%d)

# Restore
sudo systemctl stop [project].service
cp [project].db.backup-YYYYMMDD [project].db
sudo systemctl start [project].service
```
