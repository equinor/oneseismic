# Run manually to create base resources structure
# (run once or when structure update is needed)

# Required parameters

# resource group
RESOURCE_GROUP=<resource group>
# prefix for all created resources
SETUP_PREFIX=ci
# test or prod
ENVIRONMENT=test

# (Data of service principal who is allowed to create resources in RESOURCE_GROUP)
SERVICE_PRINCIPAL_CLIENT_ID=<service principal client/application id>
SERVICE_PRINCIPAL_SECRET=<service principal secret value>
ENTERPRISE_APPLICATION_OBJECT_ID=<enterprise application object id>

# Consts and vars

DEPLOYMENT_NAME=infrastructureSetup
NGINX_IMAGE_PATH='support/nginx'


# requires permissions to assign roles in Azure RBAC (e.g. Resource Group Owner)
az login

az deployment group create \
  --name $DEPLOYMENT_NAME \
  --resource-group $RESOURCE_GROUP \
  --template-file support_structure.bicep \
  --parameters \
      setupPrefix=$SETUP_PREFIX \
      environment=$ENVIRONMENT \
      principalId=$ENTERPRISE_APPLICATION_OBJECT_ID

registryURL=$(az deployment group show \
  -g $RESOURCE_GROUP \
  -n $DEPLOYMENT_NAME \
  --query properties.outputs.containerRegistryURL.value \
  -o tsv
  )

# push nginx image used for path-based routing
echo "$SERVICE_PRINCIPAL_SECRET" | \
  docker login $registryURL -u $SERVICE_PRINCIPAL_CLIENT_ID --password-stdin
docker build -f nginx.Dockerfile -t $registryURL/$NGINX_IMAGE_PATH .
docker push $registryURL/$NGINX_IMAGE_PATH
