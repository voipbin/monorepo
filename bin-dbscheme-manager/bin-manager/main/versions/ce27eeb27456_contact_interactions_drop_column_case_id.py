"""contact_interactions_drop_column_case_id

Revision ID: ce27eeb27456
Revises: a10299e7932a
Create Date: 2026-07-11 07:27:13.414482

Drops the dead case_id FK from contact_interactions (VOIP-1245).

case_id was added by 8d5a344905e7 (VOIP-1228) to link an Interaction to
the Case it was projected into. Design VOIP-1243 §7 later removed
automatic Case creation as a side effect of Interaction projection --
Case creation is now exclusively an explicit action (Flow action
case_create / AI tool case_create). Since that change,
Interaction.CaseID is never written after INSERT (grep across the
monorepo confirms no UPDATE path exists), so every row projected since
VOIP-1243 has case_id permanently NULL. The Go field was removed in the
same change as this migration; the column is not exposed in the OpenAPI
spec, so this is not an external API break.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ce27eeb27456'
down_revision = 'a10299e7932a'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""DROP INDEX idx_contact_interactions_case_id ON contact_interactions;""")
    op.execute("""ALTER TABLE contact_interactions DROP COLUMN case_id;""")


def downgrade():
    op.execute("""ALTER TABLE contact_interactions ADD COLUMN case_id BINARY(16) DEFAULT NULL;""")
    op.execute("""CREATE INDEX idx_contact_interactions_case_id ON contact_interactions(case_id);""")
