# Remove the associated data from secretcrypto

## Change

`bin-common-handler/pkg/secretcrypto` encrypted a stored secret with AES-256-GCM and optional
associated data (AAD) that bound a ciphertext to its row. The AAD is removed: `Encrypt(plaintext)` and
`Decrypt(blob, version)` no longer take it, and `AAD()` and `AADLiteral()` are deleted. The stored
value is `gcm.Seal(nonce, nonce, plaintext, nil)`: byte-identical to the former no-AAD mode, which the
ai-manager layouts already used. The algorithm, the key list format, the master key environment
variables and the layout adapters (`JoinSplit`, `SplitBlob`, `PackVersioned`, `UnpackVersioned`) do not
change.

Consumers: `confighandler` (the `table` argument of `NewHandler` and the handler field existed only for
the AAD and are removed), `bin-email-manager` (`emailconfig`) and `bin-number-manager`
(`providerhandler`: the provider name argument of `encryptSecret` and `decryptSecret` is removed).

## Compatibility

There is no compatibility read path. A ciphertext written with the associated data fails to decrypt
(`ErrDecrypt`): number-manager reports the provider as `db_undecryptable` and does not use the
environment credentials, and email-manager reports the key as undecryptable. The stored secret must be
entered again with the new image (`PUT /v1/number_providers/{name}/secret`, `email-control config
set`). Re-entering overwrites the row without decrypting it. A ciphertext is no longer bound to its row:
any row written under the same key version decrypts.

Order: the secret can be entered again only after every instance runs the new image, because an
instance that still runs the earlier version cannot read a row written by the new one. Rolling the
image back makes the re-entered rows unreadable again.

## Verification

- The fixed vectors of `secretcrypto` and number-manager were sealed with the standard library alone
  with no associated data; the library opens them, and the standard library opens what the library
  writes.
- A vector sealed with associated data fails to decrypt in `secretcrypto`, `confighandler`
  (`db_undecryptable`, no environment fallback) and `providerhandler`.
