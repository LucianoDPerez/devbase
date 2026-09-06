# SQL

Parameterize everything — string-built queries are injection bugs. Explicit
columns, never `SELECT *` in production code. Migrations up AND down, reviewed
like code. Index what you filter and join; `EXPLAIN` before blaming the ORM.
