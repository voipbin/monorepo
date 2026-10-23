# Public repository hygiene

This repository is public. Anything committed or posted here is visible to everyone, including documents, plans, PR and issue text, review comments, commit messages, branch names, code comments, test fixtures, screenshots, and pasted logs or command output. Keep the security level of this repository at the highest level at all times: prevent exposure at the time of writing, and check before you push.

## What must not appear

| Class | Examples | What to write instead |
|---|---|---|
| Personal data | customer or person names, email addresses, phone numbers, customer or account IDs, account balance or status, contact details you do not intend to publish that are embedded in image or PDF metadata | `customer A`, `customer B` (the same entity keeps the same label), a clearly fake ID |
| Credentials | passwords, API keys, tokens, private keys, `user:password@host` connection strings, service account files, `.env` and other configuration or CI files that contain them | an obviously fake value; read real values from the environment |
| Real infrastructure | IP addresses, host names, database addresses and ports of real environments | `10.0.0.x` (an arbitrary private-range example, not a real address), `203.0.113.x` (documentation range, RFC 5737), `<db-host>`, `<container-ip>` |
| Authentication and access-control configuration details, unfixed weaknesses | how access is configured in a real environment, where a known weakness is not yet fixed | leave it out; describe only what the change does. General security design and expectations (see [docs/conventions/security.md](../conventions/security.md)) are fine |
| Internal process notes | who approved or decided what, internal tooling and bookkeeping, URLs of internal systems or tickets | keep the technical rationale, drop the process detail |
| Logs, output, images | pasted production logs, stack traces, `curl` output, screenshots, captured files | redact or replace the values first; prefer a minimal synthetic reproduction |

## Before you push or open a PR

1. Search everything you are about to publish for each class above: the diff of your branch, `git diff origin/main...HEAD`, and your PR text, which is not part of the diff. Run one search per class by piping the diff into it, for example `git diff origin/main...HEAD | grep -nE '...'`. These patterns are starting points and produce false positives; review every hit:
   - email addresses: `grep -nE '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+'`
   - IPv4 addresses: `grep -nE '([0-9]{1,3}\.){3}[0-9]{1,3}'`
   - private keys: `grep -nE 'BEGIN [A-Z ]*PRIVATE KEY'`
   - credentials: `grep -niE '(password|secret|token|api[_-]?key)'`

   Names, phone numbers, and host names have no reliable pattern, so read the diff for them. Look for the classes, not only for values you already know.
2. For every credential-like string, check whether it is random-looking (UUID-shaped, long hex, mixed-case token). A random-looking value is a real credential, not a placeholder.
3. Keep distinct fixture values distinct so tables and examples stay coherent.
4. Wrap placeholders such as `<db-host>` in backticks. Outside code they render as HTML tags and disappear on GitHub.
5. Write the PR title, body, and commit message to describe the change only.

## If you find an exposure

If a credential was pushed, treat it as compromised: revoke or rotate it right away. Deleting the file, the commit, or the PR does not make it safe again. Then report it privately to support@voipbin.net with the file and line but without repeating the value. Do not open a public issue and do not describe it in a PR or commit message.

## Related

- [docs/conventions/security.md](../conventions/security.md): secrets in code, validation, and authorization rules for the code itself
