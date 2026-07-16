# Gate evidence

Store one immutable JSON document per completed compatibility gate in this
directory. A minimal document has this form:

```json
{
  "schema": 1,
  "gate": "legacy-state-replay-differential",
  "status": "passed",
  "testedCommit": "0123456789abcdef0123456789abcdef01234567",
  "source": "https://github.com/example/repository/actions/runs/123456789"
}
```

Add the file's SHA-256 digest and repository-relative path to `gateResults` in
`../stack-lock.json`. Do not add passing evidence until the named test has
actually completed against the exact pinned commit.
