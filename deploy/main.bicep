@description('Type of configuration')
@allowed([
  'test'
  'prod'
])
param environment string

@description('Setup ID: a unique prefix for resource names.')
param setupPrefix string

@description('Id of revision in the container registry')
param revisionId string = 'latest'

@description('Sign key used to verify communication between services')
@minLength(12)
@secure()
param signKey string

@description('Name of oneseismic image in the container registry')
param containerImageName string = 'apps/oneseismic'

param location string = resourceGroup().location

/**
  * All the dependent existing resources.
  * At the moment all the below are expected in current resourceGroup.
  */
@description('Storage account with the seismic data.')
param storageResourceName string = '${setupPrefix}0storage'
@description('Container registry where server images are stored.')
param containerRegistryResourceName string = '${setupPrefix}0containerRegistry'
@description('Redis cache used for storing downloaded data, application values and inter-process messages.')
param redisCacheResourceName string = '${setupPrefix}0redis'
@description('Container Apps environment to which applications are associated')
param containerAppsEnvironmentResourceName string = '${setupPrefix}0environment'

module fetchContainerApp 'fetch.bicep' = {
  name: 'fetchContainerApp'
  params: {
    environment: environment
    setupPrefix: setupPrefix
    revisionId: revisionId
    containerImageName: containerImageName
    location: location
    containerAppsEnvironmentResourceName: containerAppsEnvironmentResourceName
    redisCacheResourceName: redisCacheResourceName
    containerRegistryResourceName: containerRegistryResourceName
  }
}

// TODO: split into query and result when possible
module nginxContainerApp 'nginx.bicep' = {
  name: 'nginxContainerApp'
  params: {
    environment: environment
    setupPrefix: setupPrefix
    revisionId: revisionId
    signKey: signKey
    containerImageName: containerImageName
    location: location
    containerAppsEnvironmentResourceName: containerAppsEnvironmentResourceName
    redisCacheResourceName: redisCacheResourceName
    containerRegistryResourceName: containerRegistryResourceName
    storageResourceName: storageResourceName
  }
}

// TODO: do we need gc?

output serverURL string = nginxContainerApp.outputs.serverURL
