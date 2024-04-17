"""add table conversation_medias

Revision ID: caae1605498b
Revises: f7022d8495bc
Create Date: 2022-06-14 13:48:28.661920

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'caae1605498b'
down_revision = 'f7022d8495bc'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table conversation_medias(
            -- identity
            id            binary(16),     -- id
            customer_id   binary(16),     -- customer id

            type      varchar(255),
            Filename  varchar(2047),

            -- timestamps
            tm_create   datetime(6),  --
            tm_update   datetime(6),  --
            tm_delete   datetime(6),  --

            primary key(id)
        );
    """)
    op.execute("""create index idx_conversation_medias_customerid on conversation_medias(customer_id);""")

    op.execute("""alter table conversation_messages add column text text after data;""")
    op.execute("""alter table conversation_messages add column medias json after text;""")
    op.execute("""alter table conversation_messages drop column data;""")


def downgrade():
    op.execute("""drop table conversation_medias;""")

    op.execute("""alter table conversation_messages drop column text;""")
    op.execute("""alter table conversation_messages drop column medias;""")
    op.execute("""alter table conversation_messages add column data blob after source_target;""")


