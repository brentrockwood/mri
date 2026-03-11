# mri

[One or two sentences describing what this project does and why it exists.]

## Features

- [Feature one]
- [Feature two]
- [Feature three]

## Future Enhancements

- [Planned enhancement]

## Requirements

- [Runtime and version, e.g. Python 3.12+, Node 20+]
- [Any external services or credentials required]

## Setup

### 1. Clone the Repository

```bash
git clone https://github.com/[username]/[project].git
cd [project]
```

### 2. Create Environment

```bash
# Python example
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

# Node example
npm install
```

### 3. Configure Secrets

```bash
cp secrets.env.example secrets.env
# Edit secrets.env and fill in your values
```

See `secrets.env.example` for all available configuration options with descriptions.

### 4. First Run

```bash
# Example
python main.py run --dry-run
```

## Usage

### Basic Usage

```bash
# Example commands
python main.py run
python main.py run --limit 50
python main.py stats
```

### Dry Run Mode

Test without making changes:

```bash
python main.py run --dry-run
# or set DRY_RUN=true in secrets.env
```

### Configuration

All configuration is via environment variables in `secrets.env`. See `secrets.env.example` for comprehensive documentation.

## Development

### Run All Checks

```bash
./scripts/run_checks.sh
```

This runs security scanning, formatting, linting, type checking, and tests in sequence.

### Individual Checks

```bash
pytest                          # tests
black src/ tests/               # formatting
ruff check src/ tests/          # linting
mypy src/                       # type checking
./scripts/security_scan.sh      # secrets scan
```

### Automated Runs

For unattended/production operation, see [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

## Project Structure

```
[project]/
├── src/                    # Application source
├── tests/                  # Test files
├── scripts/
│   ├── run_checks.sh       # Run all quality checks
│   └── security_scan.sh    # Secrets and credential scanning
├── docs/
│   └── DEPLOYMENT.md       # Production deployment guide
├── project/                # Project planning and work log (not user-facing)
│   ├── doa.md              # Development Operating Agreement
│   ├── project.md          # Implementation plan (write-locked)
│   ├── context.md          # Session work log
│   └── scripts/            # Context management utilities
├── .env.example            # Non-secret config defaults
├── secrets.env.example     # Secret config template (copy to secrets.env)
└── secrets.env             # Your secrets — never commit this
```

## Security

- **Never commit** `secrets.env` or credential files — they are gitignored by default
- Run `./scripts/security_scan.sh` before every push (also run by `./scripts/run_checks.sh`)
- See `secrets.env.example` for what belongs in secrets vs. `.env`

## Troubleshooting

### [Common issue title]

[Description and fix.]

### [Another common issue]

[Description and fix.]
