"""transcribe_transcripts_replace_tm_transcript_with_offset_ms

Revision ID: 2ebeae4fd8b5
Revises: 79119e39e511
Create Date: 2026-09-16

VOIP-1528: transcribe_transcripts.tm_transcript 컬럼을 offset_ms 로 대체한다.

기존 tm_transcript(datetime(6)) 컬럼은 "전사 세그먼트의 발화 시작 오프셋"을
연도 0001 기준 시각(time.Date(1,1,1,...).Add(offset))으로 인코딩해 저장하는
타입 오용 상태였다. 이 값은 MySQL/MariaDB DATETIME 유효범위(1000~9999) 밖이라
비-strict sql_mode 에서 zero-value(0000-00-00 00:00:00.000000)로 치환 저장되고,
읽을 때 Go time.Parse 가 "month out of range" 로 실패해 transcript scan 자체가
깨졌다(2026-02-05 string->time 마이그레이션 이후 표면화).

본 마이그레이션은 tm_transcript 를 drop 하고, 발화 오프셋을 밀리초 정수로 담는
offset_ms(bigint) 컬럼을 추가한다. 프로덕션에 보존 대상 데이터가 없음을 사전
확인했으므로(과거 transcript 는 빈 배열, 장애 row 는 조회 500) 데이터 백필은
불필요하다.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '2ebeae4fd8b5'
down_revision = '79119e39e511'
branch_labels = None
depends_on = None


def _column_exists(conn, table, column):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.columns "
        "WHERE table_schema = DATABASE() AND table_name = :table AND column_name = :col"
    ), {'table': table, 'col': column})
    return result.scalar() > 0


def upgrade():
    conn = op.get_bind()
    if _column_exists(conn, 'transcribe_transcripts', 'tm_transcript'):
        op.execute("""alter table transcribe_transcripts drop column tm_transcript;""")
    if not _column_exists(conn, 'transcribe_transcripts', 'offset_ms'):
        op.execute("""alter table transcribe_transcripts add column offset_ms bigint after message;""")


def downgrade():
    conn = op.get_bind()
    if _column_exists(conn, 'transcribe_transcripts', 'offset_ms'):
        op.execute("""alter table transcribe_transcripts drop column offset_ms;""")
    if not _column_exists(conn, 'transcribe_transcripts', 'tm_transcript'):
        op.execute("""alter table transcribe_transcripts add column tm_transcript datetime(6) after message;""")
