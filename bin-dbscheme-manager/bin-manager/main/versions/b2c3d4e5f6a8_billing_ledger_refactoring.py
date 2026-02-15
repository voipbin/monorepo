"""billing_ledger_refactoring

Revision ID: b2c3d4e5f6a8
Revises: a1b2c3d4e5f7
Create Date: 2026-02-15 20:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'b2c3d4e5f6a8'
down_revision = 'a1b2c3d4e5f7'
branch_labels = None
depends_on = None


def upgrade():
    # =========================================================================
    # 1. billing_billings: Add new ledger columns
    # =========================================================================
    op.add_column('billing_billings',
                  sa.Column('transaction_type', sa.String(32), server_default='usage', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('usage_duration', sa.Integer(), server_default='0', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('billable_units', sa.Integer(), server_default='0', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('rate_token_per_unit', sa.BigInteger(), server_default='0', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('rate_credit_per_unit', sa.BigInteger(), server_default='0', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('amount_token', sa.BigInteger(), server_default='0', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('amount_credit', sa.BigInteger(), server_default='0', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('balance_token_snapshot', sa.BigInteger(), server_default='0', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('balance_credit_snapshot', sa.BigInteger(), server_default='0', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('idempotency_key', sa.LargeBinary(16)))

    # =========================================================================
    # 2. billing_billings: Migrate existing data from old cost columns to new
    # =========================================================================
    op.execute("""
        UPDATE billing_billings SET
            billable_units = CEIL(cost_unit_count),
            usage_duration = COALESCE(
                TIMESTAMPDIFF(SECOND, tm_billing_start, tm_billing_end), 0
            ),
            rate_token_per_unit = cost_token_per_unit,
            rate_credit_per_unit = CAST(cost_credit_per_unit * 1000000 AS SIGNED),
            amount_token = -cost_token_total,
            amount_credit = -CAST(cost_credit_total * 1000000 AS SIGNED)
    """)

    # =========================================================================
    # 3. billing_billings: Drop old cost columns
    # =========================================================================
    op.drop_column('billing_billings', 'cost_unit_count')
    op.drop_column('billing_billings', 'cost_token_per_unit')
    op.drop_column('billing_billings', 'cost_token_total')
    op.drop_column('billing_billings', 'cost_credit_per_unit')
    op.drop_column('billing_billings', 'cost_credit_total')

    # =========================================================================
    # 4. billing_accounts: Add new columns for State+Ledger pattern
    # =========================================================================
    op.add_column('billing_accounts',
                  sa.Column('balance_credit', sa.BigInteger(), server_default='0', nullable=False))
    op.add_column('billing_accounts',
                  sa.Column('balance_token', sa.BigInteger(), server_default='0', nullable=False))
    op.add_column('billing_accounts',
                  sa.Column('tm_last_topup', sa.DateTime(timezone=False), nullable=True))
    op.add_column('billing_accounts',
                  sa.Column('tm_next_topup', sa.DateTime(timezone=False), nullable=True))

    # =========================================================================
    # 5. billing_accounts: Convert balance float to balance_credit int64 (micros)
    # =========================================================================
    op.execute("""
        UPDATE billing_accounts SET
            balance_credit = CAST(balance * 1000000 AS SIGNED)
    """)

    # =========================================================================
    # 6. billing_accounts: Snapshot allowance tokens into balance_token
    # =========================================================================
    op.execute("""
        UPDATE billing_accounts a
        LEFT JOIN billing_allowances al ON a.id = al.account_id
            AND al.cycle_start <= NOW()
            AND al.cycle_end > NOW()
            AND al.tm_delete IS NULL
        SET a.balance_token = COALESCE(al.tokens_total - al.tokens_used, 0)
    """)

    # =========================================================================
    # 7. billing_accounts: Drop old balance column
    # =========================================================================
    op.drop_column('billing_accounts', 'balance')

    # =========================================================================
    # 8. Drop billing_allowances table (no longer needed)
    # =========================================================================
    op.drop_table('billing_allowances')


def downgrade():
    # =========================================================================
    # 1. Recreate billing_allowances table
    # =========================================================================
    op.execute("""
        CREATE TABLE IF NOT EXISTS billing_allowances (
            id              BINARY(16) PRIMARY KEY,
            customer_id     BINARY(16) NOT NULL,
            account_id      BINARY(16) NOT NULL,
            cycle_start     DATETIME(6) NOT NULL,
            cycle_end       DATETIME(6) NOT NULL,
            tokens_total    INT NOT NULL,
            tokens_used     INT NOT NULL DEFAULT 0,
            tm_create       DATETIME(6) NOT NULL,
            tm_update       DATETIME(6) NOT NULL,
            tm_delete       DATETIME(6) NULL,
            INDEX idx_billing_allowances_customer_id (customer_id),
            INDEX idx_billing_allowances_account_id (account_id),
            UNIQUE INDEX idx_billing_allowances_account_cycle (account_id, cycle_start)
        );
    """)

    # =========================================================================
    # 2. billing_accounts: Restore old balance column
    # =========================================================================
    op.add_column('billing_accounts',
                  sa.Column('balance', sa.Float(), server_default='0', nullable=False))

    # Convert balance_credit (micros) back to balance (float)
    op.execute("""
        UPDATE billing_accounts SET
            balance = balance_credit / 1000000.0
    """)

    # Drop new columns
    op.drop_column('billing_accounts', 'balance_credit')
    op.drop_column('billing_accounts', 'balance_token')
    op.drop_column('billing_accounts', 'tm_last_topup')
    op.drop_column('billing_accounts', 'tm_next_topup')

    # =========================================================================
    # 3. billing_billings: Restore old cost columns
    # =========================================================================
    op.add_column('billing_billings',
                  sa.Column('cost_unit_count', sa.Float(), server_default='0', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('cost_token_per_unit', sa.Integer(), server_default='0', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('cost_token_total', sa.Integer(), server_default='0', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('cost_credit_per_unit', sa.Float(), server_default='0', nullable=False))
    op.add_column('billing_billings',
                  sa.Column('cost_credit_total', sa.Float(), server_default='0', nullable=False))

    # Convert new columns back to old format
    op.execute("""
        UPDATE billing_billings SET
            cost_unit_count = billable_units,
            cost_token_per_unit = rate_token_per_unit,
            cost_credit_per_unit = rate_credit_per_unit / 1000000.0,
            cost_token_total = -amount_token,
            cost_credit_total = -amount_credit / 1000000.0
    """)

    # Drop new ledger columns
    op.drop_column('billing_billings', 'transaction_type')
    op.drop_column('billing_billings', 'usage_duration')
    op.drop_column('billing_billings', 'billable_units')
    op.drop_column('billing_billings', 'rate_token_per_unit')
    op.drop_column('billing_billings', 'rate_credit_per_unit')
    op.drop_column('billing_billings', 'amount_token')
    op.drop_column('billing_billings', 'amount_credit')
    op.drop_column('billing_billings', 'balance_token_snapshot')
    op.drop_column('billing_billings', 'balance_credit_snapshot')
    op.drop_column('billing_billings', 'idempotency_key')
