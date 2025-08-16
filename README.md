# Ad Targeting Engine

A **read-optimized delivery microservice** that returns matching campaigns for a request (`app`, `country`, `os`).  
Designed for **billions of requests** with only thousands of campaigns, using:
- **Lock-free snapshots** with inverted indexes
- **Postgres LISTEN/NOTIFY** for live DB refresh
- **Prometheus metrics** for observability
- **Chi middleware** for HTTP routing

---

##  Overview

This service powers a simple **Targeting Engine** that:
- Accepts requests with `app`, `country`, and `os`.
- Matches them against campaigns and targeting rules stored in a DB.
- Serves only **active campaigns**.
- Uses in-memory snapshots for **ultra-fast reads**.

---

## Run

### Local
```bash
make run
```

### Docker
```bash
make docker-build && make docker-run
```

### Config
Configs are stored in `env/` (`application.yaml`, `application-dev.yaml`, etc).  
Update DB connection + cache settings as needed.

---

## Database

- Postgres is required.  
- Schema migration is in `db/migrations/001_initial_schema.up.sql`.  
- Run migrations before starting service:
```bash
psql -U postgres -d targeting_engine -f db/migrations/001_initial_schema.up.sql
```

---

## API Usage

### Endpoint
```
GET /v1/delivery?app={app}&country={country}&os={os}
```

### Examples

**Valid request, match found**
```
GET /v1/delivery?app=com.abc.xyz&country=germany&os=android
```
Response `200 OK`:
```json
[
  {
    "cid": "duolingo",
    "img": "https://somelink2",
    "cta": "Install"
  }
]
```
