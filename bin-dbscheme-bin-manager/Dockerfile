FROM python:3.9.1


# Install mysqldb
RUN pip3 install mysqlclient

# Install alembic
RUN pip3 install alembic

# Copy service accounts
COPY bin-manager bin-manager

# Run alembic
CMD cd bin-manager && alembic -c alembic.ini upgrade head
