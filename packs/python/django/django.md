# Django

ORM first: `select_related`/`prefetch_related` against N+1, raw SQL only with
justification. Validation in forms/serializers, never in views alone. No logic
in templates beyond presentation. Named, reversible migrations. `DEBUG = False`
with `ALLOWED_HOSTS` set anywhere near production.
