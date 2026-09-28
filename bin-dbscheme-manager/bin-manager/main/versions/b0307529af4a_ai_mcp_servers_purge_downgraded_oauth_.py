"""ai_mcp_servers_purge_downgraded_oauth_tokens

Revision ID: b0307529af4a
Revises: 9a85d4662372
Create Date: 2026-09-28 12:57:51.510266

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'b0307529af4a'
down_revision = '9a85d4662372'
branch_labels = None
depends_on = None


# WHY THIS EXISTS
#   Moving an MCP server out of auth_type 'oauth' was always permitted on PUT,
#   and until now it wrote auth_type alone. The vendor access and refresh
#   tokens stayed in the row, still decryptable, on a server that no longer
#   claims to be an OAuth connection. mcpserverhandler.Update now erases them
#   on that transition, but rows downgraded BEFORE that change still hold them.
#
#   Those rows are worse off than the soft-deleted ones the previous revision
#   cleaned up, because no customer gesture erases them: the admin UI renders
#   the auth type as a fixed badge once a server is connected, and there is no
#   disconnect endpoint, so the only path that ever zeroed tokens was deleting
#   the server outright. A customer who switched a connected server to a static
#   credential reasonably believes the OAuth connection is gone. It is not.
#
#   The rows also misreport themselves: has_secret is derived from
#   secret_ciphertext OR access_token_ciphertext
#   (bin-ai-manager/pkg/dbhandler/mcpserver.go), so a bearer row carrying a
#   stale access token advertises a stored credential that the bearer path
#   never reads. Every tool call against it fails to decrypt, with nothing in
#   the API to explain why. Clearing the token columns makes has_secret tell
#   the truth.
#
#   Scope differs from the previous revision in both directions, deliberately:
#     - Only LIVE rows (tm_delete IS NULL). Deleted rows were handled by
#       9a85d4662372.
#     - Only rows that are NOT oauth. A connected server must keep its tokens.
#     - The secret envelope (secret_ciphertext, secret_nonce, key_version) is
#       NOT touched: on these rows it holds the static credential the customer
#       switched TO, which is in active use. Only the OAuth envelope goes.
#     - oauth_vendor IS cleared here, unlike in 9a85d4662372. On a deleted row
#       it is retained as audit metadata; on a live non-oauth row it is a false
#       statement, since it is published as set only while auth_type is oauth.
#
#   Keep this column set in sync with the clearing block in
#   bin-ai-manager/pkg/mcpserverhandler/handler.go (Update).
#
# RECOVERY (if upgrade fails partway):
#   A single UPDATE, so MySQL either applies it or rolls it back; there is no
#   half-migrated state to repair. If it fails for an unrelated reason (lock
#   timeout, connection loss), re-run `alembic upgrade head`: the trailing
#   predicate only matches rows that still hold an OAuth artifact, so a re-run
#   after a successful run matches nothing.
#
#   To inspect the affected rows before or after:
#     SELECT COUNT(*) FROM ai_mcp_servers
#      WHERE tm_delete IS NULL
#        AND (auth_type IS NULL OR auth_type <> 'oauth')
#        AND (access_token_ciphertext  IS NOT NULL
#          OR access_token_nonce       IS NOT NULL
#          OR refresh_token_ciphertext IS NOT NULL
#          OR refresh_token_nonce      IS NOT NULL
#          OR (oauth_vendor IS NOT NULL AND oauth_vendor <> ''));


def upgrade():
    # auth_type IS NULL is included because the column is nullable and an
    # absent value means "no auth", which is not oauth. Comparing with <>
    # alone would skip those rows, since NULL <> 'oauth' is NULL, not true.
    op.execute("""
        UPDATE ai_mcp_servers
          SET oauth_vendor             = '',
              access_token_ciphertext  = NULL,
              access_token_nonce       = NULL,
              refresh_token_ciphertext = NULL,
              refresh_token_nonce      = NULL
          WHERE tm_delete IS NULL
            AND (auth_type IS NULL OR auth_type <> 'oauth')
            AND (access_token_ciphertext  IS NOT NULL
              OR access_token_nonce       IS NOT NULL
              OR refresh_token_ciphertext IS NOT NULL
              OR refresh_token_nonce      IS NOT NULL
              OR (oauth_vendor IS NOT NULL AND oauth_vendor <> ''))
    """)


def downgrade():
    # Intentionally a no-op, and not an omission.
    #
    # The upgrade destroys ciphertext rather than moving it, so there is
    # nothing to restore -- and restoring third-party tokens the customer
    # already walked away from would be the wrong outcome even if it were
    # possible. Downgrading past this revision leaves the affected rows
    # cleared, which is the state the downgrade was always supposed to produce.
    #
    # A customer who wants OAuth back re-runs the authorization flow, which
    # repopulates every one of these columns.
    pass
