
# Usage: tools to cleanup the artifact without tag retention policy (keep the image pushed in last week)

## Prerequest


1. If the database is in the k8s cluster, forward the db container's 5432 to localhost


```
docker build -t firstfloor/cleanup_artifact:1.0 .
docker run -v firstfloor/cleanup_artifact:1.0 cleanup_artifact --help
Usage of cleanup_artifact:
  -db_host string
        Postgres database host (default "localhost")
  -db_name string
        Postgres database name (default "registry")
  -db_pass string
        Postgres database password (default "root123")
  -db_port int
        Postgres database port (default 5432)
  -db_user string
        Postgres database user (default "postgres")
  -dry_run
        Whether to skip deleting files
  -harbor_host string
        Harbor host (default "10.202.250.197")
  -harbor_pass string
        Harbor password (default "Harbor12345")
  -harbor_user string
        Harbor user (default "admin")
  -sql_condition string
        SQL condition, empty or like '-sql_condition="p.name = 'tkg%' and r.name like 'tkg/sandbox/%'"

```

Usage example:

```
    go run . -db_host=10.202.250.197 -harbor_host=10.202.250.197 -weeks=2 -dry_run=true
```


