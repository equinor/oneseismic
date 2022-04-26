@description('Type of configuration')
@allowed([
  'test'
  'prod'
])
param environment string

@description('Setup ID: a unique prefix for resource names.')
param setupPrefix string

@description('Id of revision in the container registry')
param revisionId string

@description('Name of oneseismic image in the container registry')
param containerImageName string = 'apps/oneseismic'

param location string = resourceGroup().location

/*
 * Could be changed to parameters in future.
 * One Redis instance could be reused for different flows if properly
 * parametrized in other services. 
 */
var redisStream = 'jobs'
var redisConsumerGroup = 'fetch'

/*
 * All the dependent existing resources.
 * At the moment all the below are expected in current resourceGroup.
 */
@description('Container registry where server images are stored.')
param containerRegistryResourceName string
@description('Redis cache used for storing downloaded data, application values and inter-process messages.')
param redisCacheResourceName string
@description('Container Apps environment to which applications are associated')
param containerAppsEnvironmentResourceName string

var containerCPU = {
  // workaround: values are supposed to be floats, but bicep doesn't support it
  prod: json('0.5')
  test: json('0.5')
}
var containerMemory = {
  prod: '1Gi'
  test: '1Gi'
}

var minReplicas = {
  prod: 2
  test: 1
}

var maxReplicas = {
  prod: 10
  test: 1
}

var scaleOnPendingEntriesCount = {
  prod: '10'
  test: '100'
}

/* Note: explicit passing of resource objects is in development, so
 * it would be possibe to define them once in the main file and
 * pass the resources themselves.
 */
resource containerAppsEnvironment 'Microsoft.App/managedEnvironments@2022-01-01-preview' existing = {
  name: containerAppsEnvironmentResourceName
}

resource containerRegistry 'Microsoft.ContainerRegistry/registries@2021-09-01' existing = {
  name: containerRegistryResourceName
}

resource redisCache 'Microsoft.Cache/Redis@2019-07-01' existing = {
  name: redisCacheResourceName
}

var fetchAppName = '${setupPrefix}-fetch'
var redisAddress = '${redisCache.properties.hostName}:${redisCache.properties.sslPort}'
var imagePath = '${containerRegistry.properties.loginServer}/${containerImageName}:${revisionId}'

/* Existing bicep anti-code-duplication capabilities will be only confusing here,
 * so all resources are defined explicitly.
 */

resource fetchContainerApp 'Microsoft.App/containerApps@2022-01-01-preview' = {
  name: fetchAppName
  location: location
  properties: {
    managedEnvironmentId: containerAppsEnvironment.id
    configuration: {
      activeRevisionsMode: 'single'
      secrets: [
        {
          name: 'registry-password'
          value: containerRegistry.listCredentials().passwords[0].value
        }
        {
          name: 'redis-password'
          value: redisCache.listKeys().primaryKey
        }
      ]
      registries: [
        {
          // issue 153, containerRegistry.properties.loginServer expected
          server: '${toLower(containerRegistry.name)}.azurecr.io'
          username: containerRegistry.listCredentials().username
          passwordSecretRef: 'registry-password'
        }
      ]
    }
    template: {
      revisionSuffix: revisionId
      containers: [
        {
          name: fetchAppName
          image: imagePath
          resources: {
            cpu: containerCPU[environment]
            memory: containerMemory[environment]
          }
          env: [
            {
              name: 'REDIS_URL'
              value: redisAddress
            }
            {
              name: 'REDIS_PASSWORD'
              secretRef: 'redis-password'
            }
          ]
          command: [
            'oneseismic-fetch'
          ]
          args: [
            '--secureConnections'
          ]
        }
      ]
      scale: {
        minReplicas: minReplicas[environment]
        maxReplicas: maxReplicas[environment]
        rules: [
          {
            name: 'redis-stream-based-autoscaling'
            custom: {
              type: 'redis-streams'
              metadata: {
                address: redisAddress
                stream: redisStream
                consumerGroup: redisConsumerGroup
                pendingEntriesCount: scaleOnPendingEntriesCount[environment]
                enableTLS: 'true'
              }
              auth: [
                {
                  secretRef: 'redis-password'
                  triggerParameter: 'password'
                }
              ]
            }
          }
        ]
      }
    }
  }
}
