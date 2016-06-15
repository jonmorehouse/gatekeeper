db_setup:
	@echo "starting mysql container ..." 
	docker run \
		--name=gatekeeper-mysql_api_upstream \
		--restart=always \
		--detach \
		-p 3306:3306 \
		--volume=/gatekeeper/plugins/mysql-api-upstreams/migrations:/migrations \
		-e MYSQL_ROOT_PASSWORD=root \
		-e MYSQL_PASSWORD=gatekeeper \
		-e MYSQL_USER=gatekeeper \
		-e MYSQL_DATABASE=gatekeeper \
		mysql:latest

db_migrate:
	@echo "running migration ..."
	docker exec -it gatekeeper-mysql_api_upstream \
		sh -c "MYSQL_PWD=gatekeeper mysql --user=gatekeeper gatekeeper < /migrations/schema__001.sql"

db_client:
	@echo "connecting to mysql container ..."
	docker exec -it gatekeeper-mysql_api_upstream \
		sh -c "MYSQL_PWD=gatekeeper mysql --user=gatekeeper gatekeeper"

db_teardown: 
	@echo "stopping mysql container ..."
	docker stop \
		--time=5 \
		gatekeeper-mysql_api_upstream
	docker rm gatekeeper-mysql_api_upstream

integration:
	GOPATH=/gopath go test -tags=integration .

unit:
	GOPATH=/gopath go test .
