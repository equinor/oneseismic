# Manually upload server code to setup as "latest"
# (run every time server must be updated)

# Required parameters

# resource group
RESOURCE_GROUP=<resource group>
# prefix for all created resources
SETUP_PREFIX=ci
# test or prod
ENVIRONMENT=test
# string used as a sign key
SIGN_KEY=<long key used for authorization tokens>


# (Data of service principal who is allowed to create resources in RESOURCE_GROUP)
SERVICE_PRINCIPAL_CLIENT_ID=<service principal client/application id>
SERVICE_PRINCIPAL_SECRET=<service principal secret value>

# Consts and vars

REGISTRY_URL="$SETUP_PREFIX"0containerRegistry.azurecr.io
DEPLOYMENT_NAME=updatecode

echo "$SERVICE_PRINCIPAL_SECRET" | docker login $REGISTRY_URL -u $SERVICE_PRINCIPAL_CLIENT_ID --password-stdin
docker build -t $REGISTRY_URL/apps/oneseismic ./..
docker push $REGISTRY_URL/apps/oneseismic

# login as contributor to resource group
az login

# use more parameters if needed
az deployment group create \
  --name $DEPLOYMENT_NAME \
  --resource-group $RESOURCE_GROUP \
  --template-file main.bicep \
  --parameters setupPrefix=$SETUP_PREFIX environment=$ENVIRONMENT signKey=$SIGN_KEY
