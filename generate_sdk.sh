pushd api
go generate
popd
# java -jar ./swagger-codegen-cli-2.4.10.jar generate -l python -i api/docs/swagger.yaml -c api-codegen-config.json -o sdk/
docker run --rm -v $(pwd):/local swaggerapi/swagger-codegen-cli generate -i /local/api/docs/swagger.yaml -l python -o /local/sdk -c /local/sdk/.swagger-codegen.json 
