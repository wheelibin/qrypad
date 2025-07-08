.PHONY: testdb-pg-up testdb-pg-down testdb-mysql-up testdb-mysql-down

testdb-pg-up:
	cd test-db/postgres && docker compose up -d

testdb-pg-down:
	cd test-db/postgres && docker compose down -v

testdb-mysql-up:
	cd test-db/mysql && docker compose up -d

testdb-mysql-down:
	cd test-db/mysql && docker compose down -v

test-sqlite-up:
	sqlite3 test-db/sqlite/sqlite.db < test-db/sqlite/init.sql

