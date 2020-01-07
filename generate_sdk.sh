pushd api
swag init
popd
java -jar ./swagger-codegen-cli-2.4.10.jar generate -l python -i api/docs/swagger.yaml -c api-codegen-config.json -o sdk/
