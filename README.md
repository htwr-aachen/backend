
# HTWR Backend

This is the backend service powering small parts of htwr-aachen.de.

It is written in Go and is designed to be a clean, and my structured reference for building modern Go servers.

Services provided by the backend:

- Panikzettel: Serves the cheat sheets and summaries available on the site.

- Q&A Service: An experimental Q&A module (currently implemented but mostly unused).

- Admin UI: A server-side rendered dashboard to manage content. Templ and HTMX experiment. This is only accessible to authorized users.

Running locally

Install dependencies:

```bash
go mod download
pnpm install # used for tailwindcss
```

Generate Templates

```bash
templ generate
```

Run the server:

```bash
go run main.go
```

Although for development a quick Makefile allows for go server restarts on changes and hot templ & tailwind generation
```bash
make {templ,server,tailwind,dev} # dev runs all 3 in parallel
```

## Configuration

The backend is extensively configurable using a yaml file at the following locations (From most prioritized to Least)
- --config path
- ./htwr-backend.yaml
- $XDG_CONFIG_HOME/htwr-backend/htwr-backend.yaml
- /etc/htwr-backend/htwr-backend.yaml

We also load from the environment variables and replace each JSON path with a _ and preferably upper case i.e. 
```yaml
Database:
    DBHost: localhost
```
would be configurable via DATABASE_DBHOST env var.

For an overview over the configurables see pkg/defaults and pkg/config

