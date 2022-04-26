@description('Short name prefix for all new deployments')
param setupPrefix string

@description('Type of configuration')
@allowed([
  'test'
  //'prod' Can't be used in production until approval is granted
])
param environment string

@description('Existing service principal used to run operations')
param principalId string

var containerRegistrySKU = {
  prod: 'Basic'
  test: 'Basic'
}

var storageSKU = {
  prod: 'Standard_LRS'
  test: 'Standard_LRS'
}

var storageAccessTier = {
  prod: 'Hot'
  test: 'Cool'
}

var redisCacheSKU = {
  capacity: {
    prod: 0
    test: 0
  }
  family: {
    prod: 'C'
    test: 'C'
  }
  name: {
    prod: 'Standard'
    test: 'Basic'
  }
}

@description('Location for all deployments')
param location string = resourceGroup().location

@description('Storage account with the seismic data.')
param storageResourceName string = '${setupPrefix}0storage'
@description('Container registry where server images are stored.')
param containerRegistryResourceName string = '${setupPrefix}0containerRegistry'
@description('Redis cache used for storing downloaded data, application values and inter-process messages.')
param redisCacheResourceName string = '${setupPrefix}0redis'
@description('Log workspace for the Container Apps environment')
param logAnalyticsWorkspaceResourceName string = '${setupPrefix}0logs'
@description('Container Apps environment to which applications are associated')
param containerAppsEnvironmentResourceName string = '${setupPrefix}0environment'

resource logAnalyticsWorkspace 'Microsoft.OperationalInsights/workspaces@2021-06-01' = {
  name: logAnalyticsWorkspaceResourceName
  location: location
}

resource containerAppsEnvironment 'Microsoft.App/managedEnvironments@2022-01-01-preview' = {
  name: containerAppsEnvironmentResourceName
  location: location
  properties: {
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: logAnalyticsWorkspace.properties.customerId
        sharedKey: logAnalyticsWorkspace.listKeys().primarySharedKey
      }
    }
  }
}

resource containerRegistry 'Microsoft.ContainerRegistry/registries@2021-09-01' = {
  name: containerRegistryResourceName
  location: location
  sku: {
    name: containerRegistrySKU[environment]
  }
  // admin account is required for internal azure deployment
  properties: {
    adminUserEnabled: true
  }

  /* TODO: create an acr purge TimerTrigger task to remove
     old images from the registry as every PR would push a new one.
     Commands are still in preview and bicep examples are missing,
     so something fails and it's unclear how to invoke it yet.
   */
  // resource purgeContainersTask 'tasks@2019-04-01' = if (environment == 'test'){
  //   name: 'purgeOldContainersTask'
  //   location: location
  // }
}

// Creation of this resource requires Owner role for one who runs the deployment
resource servicePrincipalContainerRegistryPushRoleAssignment 'Microsoft.Authorization/roleAssignments@2020-04-01-preview' = {
  name: guid(containerRegistry.id, principalId)
  scope: containerRegistry
  properties: {
    description: 'Service principal would be used to push updates to registry'
    roleDefinitionId: '/providers/Microsoft.Authorization/roleDefinitions/8311e382-0749-4cb8-b61a-304f252e45ec' //AcrPush
    principalId: principalId
    principalType: 'ServicePrincipal'
  }
}

resource redisCache 'Microsoft.Cache/redis@2020-12-01' = {
  name: redisCacheResourceName
  location: location
  properties: {
    sku: {
      capacity: redisCacheSKU.capacity[environment]
      family: redisCacheSKU.family[environment]
      name: redisCacheSKU.name[environment]
    }
    redisVersion: '6'
  }
}

resource storage 'Microsoft.Storage/storageAccounts@2021-06-01' = {
  name: storageResourceName
  location: location
  sku: {
    name: storageSKU[environment]
  }
  kind: 'BlobStorage'
  properties: {
    accessTier: storageAccessTier[environment]
  }
}

output containerRegistryURL string = containerRegistry.properties.loginServer
