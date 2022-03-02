/* Once path-based routing is implemented in Container Apps, nginx wrapper can
 * be removed and file can be split into two - query and result one. It will
 * allow applications to scale separately.
 */

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

@description('Sign key used to verify communication between services')
@minLength(12)
@secure()
param signKey string

@description('Name of oneseismic image in the container registry')
param containerImageName string = 'apps/oneseismic'

@description('Name of nginx image in the container registry')
param nginxImageName string = 'support/nginx'

param location string

/*
 * All the dependent existing resources.
 * At the moment all the below are expected in current resourceGroup.
 */
@description('Storage account with the seismic data.')
param storageResourceName string
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
  prod: 0
  test: 0
}

var maxReplicas = {
  prod: 10
  test: 1
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

resource storage 'Microsoft.Storage/storageAccounts@2021-02-01' existing = {
  name: storageResourceName
}

resource redisCache 'Microsoft.Cache/Redis@2019-07-01' existing = {
  name: redisCacheResourceName
}

var nginxAppName = '${setupPrefix}-nginx'
var redisAddress = '${redisCache.properties.hostName}:${redisCache.properties.sslPort}'
var imagePath = '${containerRegistry.properties.loginServer}/${containerImageName}:${revisionId}'
var nginxImagePath = '${containerRegistry.properties.loginServer}/${nginxImageName}:latest'
// TODO: current code requires URL to end without / (fix in the code)
var storageURL = take(storage.properties.primaryEndpoints.blob, length(storage.properties.primaryEndpoints.blob) - 1)

/* Existing bicep anti-code-duplication capabilities will be only confusing here,
 * so all resources are defined explicitly.
 */

resource nginxContainerApp 'Microsoft.App/containerApps@2022-01-01-preview' = {
  name: nginxAppName
  location: location
  properties: {
    managedEnvironmentId: containerAppsEnvironment.id
    configuration: {
      activeRevisionsMode: 'single'
      ingress: {
        external: true
        targetPort: 8080
      }
      secrets: [
        {
          name: 'registry-password'
          value: containerRegistry.listCredentials().passwords[0].value
        }
        {
          name: 'redis-password'
          value: redisCache.listKeys().primaryKey
        }
        {
          name: 'sign-key'
          value: signKey
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
          name: 'nginx'
          image: nginxImagePath
          resources: {
            cpu: containerCPU[environment]
            memory: containerMemory[environment]
          }
        }
        {
          name: 'query'
          image: imagePath
          resources: {
            cpu: containerCPU[environment]
            memory: containerMemory[environment]
          }
          env: [
            {
              name: 'STORAGE_URL'
              value: storageURL
            }
            {
              name: 'REDIS_URL'
              value: redisAddress
            }
            {
              name: 'REDIS_PASSWORD'
              secretRef: 'redis-password'
            }
            {
              name: 'SIGN_KEY'
              secretRef: 'sign-key'
            }
          ]
          command: [
            'oneseismic-query'
          ]
          args: [
            '--secureConnections'
            '--port=8085'
          ]
        }
        {
          name: 'result'
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
            {
              name: 'SIGN_KEY'
              secretRef: 'sign-key'
            }
          ]
          command: [
            'oneseismic-result'
          ]
          args: [
            '--secureConnections'
            '--port=8084'
          ]
        }
      ]
      scale: {
        minReplicas: minReplicas[environment]
        maxReplicas: maxReplicas[environment]
      }
    }
  }
}

output serverURL string = 'https://${nginxContainerApp.properties.configuration.ingress.fqdn}'
