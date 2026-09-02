"""campaign_campaigns_add_index_tm_delete

Revision ID: ab1507b96589
Revises: 9f7413ad1d07
Create Date: 2026-09-02 16:33:11.673388

Adds an index on campaign_campaigns.tm_delete for the orphaned-flow
reconciliation job (VOIP-1444,
pkg/dbhandler.CampaignListDeletedSince). Without it, the job's
`WHERE tm_delete >= ? ORDER BY tm_delete DESC LIMIT ?` query
full-scans the table on every scheduled run (every 6 hours by
default -- see the paired schedule seed migration,
9f7413ad1d07). campaign_campaigns previously had indexes only on
customer_id, flow_id, outplan_id, outdial_id, and queue_id -- none
on tm_delete.

Included per the plan's conservative default (add the index
regardless of measured row count/deletion rate -- small, additive,
reversible) rather than blocking on production data this session had
no access to. See
docs/plans/2026-09-02-voip-1444-orphaned-flow-reconciliation-plan.md's
"Pre-implementation check".

Per bin-dbscheme-manager/CLAUDE.md's migration+fixture-sync rule, the
test-schema fixture
(bin-campaign-manager/scripts/database_scripts/table_campaigns.sql)
is updated in this same commit to add the matching index.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ab1507b96589'
down_revision = '9f7413ad1d07'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""CREATE INDEX idx_campaign_campaigns_tm_delete ON campaign_campaigns(tm_delete);""")


def downgrade():
    op.execute("""DROP INDEX idx_campaign_campaigns_tm_delete ON campaign_campaigns;""")
