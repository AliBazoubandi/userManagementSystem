sqlcgenerate:
	docker run --rm -v D:\userManagementSystem:/src -w /src sqlc/sqlc generate

checkdb:
	docker exec -it postgres_container psql -U alibazoubandi -d myDataBase

.PHONY: sqlcgenerate, checkdb