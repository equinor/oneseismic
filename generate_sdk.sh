java -jar ./swagger-codegen-cli-2.4.10.jar generate -l python -i docs/swagger.yaml -c api-codegen-config.json -Dapis=manifest -Dmodels=store.Manifest -DsupportingFiles -o sdk/
