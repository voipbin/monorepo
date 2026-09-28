"""ai_mcp_servers_purge_deleted_credentials

Revision ID: 9a85d4662372
Revises: 9cc9fded2d17
Create Date: 2026-09-28 11:02:43.780938

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '9a85d4662372'
down_revision = '9cc9fded2d17'
branch_labels = None
depends_on = None


# WHY THIS EXISTS
#   Deleting an MCP server is a revocation: the customer is withdrawing
#   VoIPBin's access to a third-party system. McpServerDelete now erases the
#   stored credentials in the same statement that sets tm_delete, but rows
#   soft-deleted BEFORE that change still hold their ciphertext, nonces, and
#   key_version. Those rows are unreachable through the API yet remain
#   decryptable with a key VoIPBin still holds, so the revocation the customer
#   asked for was never completed for them. This migration finishes it.
#
#   oauth_vendor and access_token_expires_at are deliberately NOT cleared:
#   they are metadata, not secrets, and are retained for audit. That matches
#   the column set McpServerDelete zeroes in
#   bin-ai-manager/pkg/dbhandler/mcpserver.go -- keep the two in sync.
#
# RECOVERY (if upgrade fails partway):
#   This is a single UPDATE, so MySQL either applies it or rolls it back; there
#   is no half-migrated state to repair. If it fails for an unrelated reason
#   (lock timeout, connection loss), simply re-run `alembic upgrade head`: the
#   statement only matches rows that still hold a credential, so a re-run after
#   a successful run matches nothing.
#
#   To inspect the affected rows before or after:
#     SELECT COUNT(*) FROM ai_mcp_servers
#      WHERE tm_delete IS NOT NULL
#        AND (secret_ciphertext IS NOT NULL
#          OR secret_nonce IS NOT NULL
#          OR access_token_ciphertext IS NOT NULL
#          OR access_token_nonce IS NOT NULL
#          OR refresh_token_ciphertext IS NOT NULL
#          OR refresh_token_nonce IS NOT NULL
#          OR key_version <> 0);


def upgrade():
    # Only soft-deleted rows. Live rows must keep their credentials -- this
    # finishes an incomplete revocation, it is not a mass revocation.
    #
    # The trailing predicate keeps the statement idempotent and keeps it from
    # rewriting rows that are already clean, so a re-run touches nothing.
    op.execute("""
        UPDATE ai_mcp_servers
          SET secret_ciphertext        = NULL,
              secret_nonce             = NULL,
              key_version              = 0,
              access_token_ciphertext  = NULL,
              access_token_nonce       = NULL,
              refresh_token_ciphertext = NULL,
              refresh_token_nonce      = NULL
          WHERE tm_delete IS NOT NULL
            AND (secret_ciphertext        IS NOT NULL
              OR secret_nonce             IS NOT NULL
              OR access_token_ciphertext  IS NOT NULL
              OR access_token_nonce       IS NOT NULL
              OR refresh_token_ciphertext IS NOT NULL
              OR refresh_token_nonce      IS NOT NULL
              OR key_version <> 0)
    """)


def downgrade():
    # Intentionally a no-op, and not an omission.
    #
    # The upgrade destroys ciphertext rather than moving it, so there is
    # nothing to restore -- and restoring revoked third-party credentials
    # would be the wrong outcome even if it were possible. Downgrading past
    # this revision leaves the affected rows cleared, which is the state the
    # delete was always supposed to produce.
    pass
