# known_hosts

Sources:

- Primary: `~/.ssh/known_hosts`
- Can be overridden via CLI `--known-hosts` (repeatable)

Parsing (MVP):

- Use the first token of each non-empty line.
- Split by commas to get multiple hosts.
- Support the bracket form `[host]:port`.
- Ignore:
  - comments (`#...`)
  - markers (`@cert-authority`, `@revoked`, ...)
  - hashed hostnames (`|1|...`) (not displayable)

`defaults.load_known_hosts`:

- `true` (default): Hosts list comes from known_hosts.
- `false`: Hosts list comes only from config (union of `[[hosts]].host` and `groups[].hosts`).
