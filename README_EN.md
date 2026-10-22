<div align="center">
  <img src="docs/public/logo-light.svg" alt="CloudCTF Logo" width="104" height="104">
  <h1>CloudCTF</h1>
  <p><strong>A self-hosted CTF competition and cyber range platform for events, education, and enterprise training</strong></p>
  <p>The player portal and admin console ship inside one Go binary, with dynamic labs, theory exams, and end-to-end event operations.</p>

  <p><a href="README.md">简体中文</a> · <strong>English</strong></p>

  <p>
    <a href="https://github.com/hexbay/CloudCTF/releases"><img src="https://img.shields.io/github/v/release/hexbay/CloudCTF?style=flat-square&logo=github&label=release" alt="Release"></a>
    <a href="https://github.com/hexbay/CloudCTF/actions/workflows/integration-tests.yml"><img src="https://img.shields.io/github/actions/workflow/status/hexbay/CloudCTF/integration-tests.yml?branch=main&style=flat-square&label=tests" alt="Integration tests"></a>
    <a href="https://github.com/hexbay/CloudCTF/actions/workflows/codeql.yml"><img src="https://img.shields.io/github/actions/workflow/status/hexbay/CloudCTF/codeql.yml?branch=main&style=flat-square&label=CodeQL" alt="CodeQL"></a>
    <a href="https://hexbay.github.io/CloudCTF/"><img src="https://img.shields.io/badge/docs-online-0f766e?style=flat-square&logo=readthedocs&logoColor=white" alt="Documentation"></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/hexbay/CloudCTF?style=flat-square" alt="License"></a>
  </p>

  <p>
    <a href="http://cloud-ctf.lostpeach.cn/"><strong>Live Demo</strong></a>
    · <a href="https://hexbay.github.io/CloudCTF/">Documentation</a>
    · <a href="https://github.com/hexbay/CloudCTF/releases">Releases</a>
    · <a href="https://github.com/hexbay/CloudCTF/issues">Issues</a>
  </p>
</div>

---

## Why CloudCTF

CloudCTF brings the player portal, admin console, competition engine, and cyber range management into one self-hosted platform. It runs from a single binary with embedded SQLite by default, while production installations can use MySQL or PostgreSQL and local or remote Docker hosts.

<table>
  <tr>
    <td width="33%" align="center"><img src="docs/public/icons/trophy.svg" width="42" alt="Competition operations"><br><strong>Competition Operations</strong><br><sub>Solo and team events, registration, scoring, leaderboards, and announcements</sub></td>
    <td width="33%" align="center"><img src="docs/public/icons/target.svg" width="42" alt="Challenge types"><br><strong>Multiple Challenge Types</strong><br><sub>Web, Pwn, Crypto, Misc, theory questions, and exams</sub></td>
    <td width="33%" align="center"><img src="docs/public/icons/docker.svg" width="42" alt="Dynamic ranges"><br><strong>Dynamic Ranges</strong><br><sub>Single and multi-container templates, resource limits, and automatic cleanup</sub></td>
  </tr>
  <tr>
    <td width="33%" align="center"><img src="docs/public/icons/library.svg" width="42" alt="Vulnerability labs"><br><strong>Vulnerability Labs</strong><br><sub>Template synchronization, CVE practice, and isolated runtime environments</sub></td>
    <td width="33%" align="center"><img src="docs/public/icons/shield.svg" width="42" alt="Access and audit"><br><strong>Access &amp; Audit</strong><br><sub>Role-based access, resource controls, audit trails, and security policies</sub></td>
    <td width="33%" align="center"><img src="docs/public/icons/chart.svg" width="42" alt="Analytics"><br><strong>Analytics</strong><br><sub>Live scores, competition data, training records, and operational views</sub></td>
  </tr>
</table>

### Highlights

- **All-in-one delivery** — the React frontend is embedded in the Go service, serving both portals from one binary.
- **Flexible deployment** — Linux AMD64 / ARM64, Docker, SQLite, MySQL, and PostgreSQL are supported.
- **Complete range lifecycle** — manage Docker hosts, images, templates, container groups, and runtimes.
- **Safe operations** — built-in backups, auditing, admin password recovery, and manual updates from GitHub Releases.
- **Open for extension** — clear APIs, modular frontend and backend code, and comprehensive documentation.

## Preview

<table>
  <tr>
    <td width="50%" align="center"><strong>Competition Management</strong></td>
    <td width="50%" align="center"><strong>Vulnerability Labs</strong></td>
  </tr>
  <tr>
    <td><img src="docs/public/images/competition.png" alt="CloudCTF competition management"></td>
    <td><img src="docs/public/images/vuln-lab-list.png" alt="CloudCTF vulnerability labs"></td>
  </tr>
</table>

## Quick Start

### Download a single binary (recommended)

Download the asset for your server from [GitHub Releases](https://github.com/hexbay/CloudCTF/releases):

| Architecture | Binary | Checksum |
| --- | --- | --- |
| Linux x86_64 | `cloudctf-linux-amd64` | `cloudctf-linux-amd64.sha256` |
| Linux ARM64 | `cloudctf-linux-arm64` | `cloudctf-linux-arm64.sha256` |

```bash
curl -LO https://github.com/hexbay/CloudCTF/releases/latest/download/cloudctf-linux-amd64
curl -LO https://github.com/hexbay/CloudCTF/releases/latest/download/cloudctf-linux-amd64.sha256
sha256sum -c cloudctf-linux-amd64.sha256
chmod +x cloudctf-linux-amd64
./cloudctf-linux-amd64
```

Once started, open:

| Entry point | URL |
| --- | --- |
| Player portal | `http://<server-ip>:8005/` |
| Admin console | `http://<server-ip>:8005/admin` |
| Health check | `http://<server-ip>:8005/healthz` |

> Development mode creates demo accounts. Configure the production environment variables before a real deployment, and do not keep demo credentials.

### Run with Docker

```bash
docker pull peach0214/cloud_ctf:latest

docker run -d --name cloudctf --restart unless-stopped \
  -p 8005:8005 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$PWD/data:/app/data" \
  -v "$PWD/static:/app/static" \
  peach0214/cloud_ctf:latest
```

The official Docker image defaults to `APP_ENV=production`. On first startup, if `data/.env` does not exist, CloudCTF copies `.env.example`, records production mode, and replaces the JWT signing key and initial admin password with secure random values. Before the first login, read the password with `grep '^FIRST_SUPERUSER_PASSWORD=' data/.env`. Keep this file private and reuse the same `data/` volume when recreating the container. `ALLOWED_ORIGINS` is not needed for same-origin access.

> Access to the Docker socket is equivalent to elevated host privileges. Isolate range nodes in production and restrict mounts, networking, and image sources.

### Database configuration

CloudCTF switches databases through the single `DATABASE_URI` setting. When it is omitted, embedded SQLite is used and the database is created at `data/cloudctf.db`. Tables and initial records are prepared automatically when an empty database is connected for the first time.

| Database | Example `DATABASE_URI` | Recommended use |
| --- | --- | --- |
| SQLite (default) | `sqlite://data/cloudctf.db` | Single-node deployments, evaluation, and small installations |
| MySQL | `mysql://cloudctf:strong-password@tcp(mysql:3306)/cloudctf?charset=utf8mb4&parseTime=True&loc=Local` | Production with existing MySQL infrastructure |
| PostgreSQL | `postgres://cloudctf:strong-password@postgres:5432/cloudctf?sslmode=disable` | Production with PostgreSQL infrastructure |

For a binary or source deployment:

```bash
export DATABASE_URI='postgres://cloudctf:strong-password@127.0.0.1:5432/cloudctf?sslmode=disable'
./cloudctf-linux-amd64
```

For Docker with an external database, add this option to the existing command:

```bash
-e DATABASE_URI='mysql://cloudctf:strong-password@tcp(mysql:3306)/cloudctf?charset=utf8mb4&parseTime=True&loc=Local'
```

For PostgreSQL:

```bash
-e DATABASE_URI='postgres://cloudctf:strong-password@postgres:5432/cloudctf?sslmode=require'
```

> Create the database before starting CloudCTF. The account needs permission to create and alter tables and indexes, and to read and write data. Inside a container, `localhost` refers to that container; use the database service name or a reachable host address. For remote PostgreSQL production deployments, use `sslmode=require` or the certificate options required by your server. Changing the URI does not migrate existing data, so back up and validate the migration before switching databases.

### Run from source

Development requires **Go 1.26+**, **Node.js 22.12+**, and npm. Dynamic labs and the full integration suite also require Docker.

```bash
git clone https://github.com/hexbay/CloudCTF.git
cd CloudCTF/frontend
npm ci
npm run build

cd ../backend
go run ./cmd/server
```

## Essential Configuration

CloudCTF reads configuration from environment variables and can also load a `.env` file selected by `CONFIG_FILE`.

| Variable | Default | Purpose |
| --- | --- | --- |
| `HOST` | `0.0.0.0` | HTTP bind address |
| `PORT` | `8005` | HTTP port |
| `APP_ENV` | source: `development`; Docker: `production` | Runtime environment and safety mode |
| `DATABASE_URI` | `sqlite://data/cloudctf.db` | SQLite, MySQL, or PostgreSQL connection |
| `REDIS_ADDR` | `redis://localhost:6379/0` | Redis URI; may include password and database number |
| `JWT_SECRET` | generated | Random JWT secret written to `data/.env` on first startup |
| `FIRST_SUPERUSER_USERNAME` | `admin` | Initial super administrator username |
| `FIRST_SUPERUSER_PASSWORD` | generated | Written to `data/.env` for the first admin login |
| `ALLOWED_ORIGINS` | empty | Configure only when the frontend uses a different origin |
| `UPDATE_ENABLED` | `true` | Enables the online update entry in the admin console |
| `UPDATE_DIR` | `data/updates` | Persistent container updates and rollback state |
| `PROXY` | empty | Shared HTTP/SOCKS5 proxy for Git repository sync and online updates |

See the [production deployment guide](https://hexbay.github.io/CloudCTF/guide/production-deploy) for complete configuration, external database, systemd, and optional Nginx reverse proxy examples. Nginx is not required; the service can be accessed directly at `http://<server-ip>:8005`.

## Operations Cheat Sheet

### Reset an administrator password

The CLI updates the database directly, revokes existing refresh sessions, and requires a password change at the next login:

```bash
./cloudctf-linux-amd64 reset-admin-password --username admin
```

For non-interactive environments, pass the password through standard input:

```bash
printf '%s\n' 'new-strong-password' | ./cloudctf-linux-amd64 reset-admin-password --username admin --password-stdin
```

### Online updates

CloudCTF checks the latest GitHub Release when a super administrator enters the admin console. It displays a notification but **never installs an update automatically**. Review the release notes and confirm it under **System Settings → System Information**. An automatic backup is created first. In containers, binaries and rollback state are persisted under `data/updates/`, survive container recreation, and roll back when the new version fails its first startup. Pull a new image when the base image or system dependencies change.

## Repository Layout

```text
CloudCTF/
├── backend/           Go API, services, database, and embedded frontend
├── frontend/          React + TypeScript player portal and admin console
├── docs/              VitePress documentation site
├── tests/integration/ HTTP end-to-end tests
├── deploy/            Deployment assets
└── .github/workflows/ CI, CodeQL, docs, and release workflows
```

## Documentation

| Guide | Covers |
| --- | --- |
| [Quick Start](https://hexbay.github.io/CloudCTF/guide/quick-start) | Download, startup, and first access |
| [Production Deployment](https://hexbay.github.io/CloudCTF/guide/production-deploy) | systemd, reverse proxies, databases, and updates |
| [Administrator Guide](https://hexbay.github.io/CloudCTF/guide/admin-guide) | Competitions, challenges, ranges, and system management |
| [Architecture](https://hexbay.github.io/CloudCTF/guide/architecture) | Modules, data flow, and development notes |
| [API Documentation](https://hexbay.github.io/CloudCTF/api/overview) | API entry points and authentication |

## Contributing

Issues and pull requests are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) first, and run at least these checks before submitting a change:

```bash
cd backend && go test ./...
cd ../frontend && npm run build
```

## Security

Do not disclose vulnerabilities through public issues. Report them privately through GitHub Security Advisories and read [SECURITY.md](SECURITY.md) for the support policy and security boundaries.

## License

CloudCTF is released under the [MIT License](LICENSE).

<div align="center">
  <sub>Built for practical security training and CTF operations.</sub>
</div>
