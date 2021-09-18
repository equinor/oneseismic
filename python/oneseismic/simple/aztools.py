"""Azure tools for oneseismic

This module makes a lot of assumptions on your behalf, but is entirely
optional, i.e. oneseismic does not depend on details implemented here. The
module does not offer many configuration options in order to keep the interface
small and simple, and to avoid simply being an obfuscation layer over the azure
libraries. If you need more flexibility or different parameters then you must
write bespoke azure integration.
"""
import datetime
from urllib.parse import urlsplit

import azure.storage.blob as azblob

class simple_blobstore_auth:
    """Simple blobstore authorization

    This class aims to be a simple and practical way of obtaining issuing SAS
    tokens, suited for small single-user programs. Its intended audience are
    oneseismic end users that don't have any particular needs, and want to get
    going right away.

    Parameters
    ----------
    resource : str
        The blob resource, e.g. 'https://<acc>.blob.core.windows.net'
    credential : azure.identity.Credential
        The azure credential, defaults to azure.identity.DefaultAzureCredential
    """
    def __init__(self, resource, credential = None):
        self.resource = resource
        self.acc = urlsplit(resource).netloc.split('.')[0]

        if credential is None:
            # only import azure.identity when it is certain we want to use it
            # importing it takes several hundred milliseconds (and from a
            # glance seems to make some http requests)
            import azure.identity
            credential = azure.identity.DefaultAzureCredential()
        self.client = azblob.BlobServiceClient(
            resource,
            credential = credential,
        )

    def user_delegation_key(self):
        """Make a new user delegation key

        Get a (possibly cached) user delegation key for making shared access
        signatures.

        You generally do not need to use this function unless you want more
        control over delegation keys, or need them for something other than
        gen_sas().

        Returns
        -------
        udk
            User delegation key for signing shared access signatures
        """

        now = datetime.datetime.now(datetime.timezone.utc)
        try:
            udk = self.cached_udk
            # azure seems to return UTC, but datetime can only parse iso
            isoexpiry = udk.signed_expiry.replace('Z', '+00:00')
            expiry = datetime.datetime.fromisoformat(isoexpiry)
            if (expiry - now) > datetime.timedelta(minutes = 5):
                return udk

        except AttributeError:
            self.cached_udk = self.client.get_user_delegation_key(
                now - datetime.timedelta(minutes = 5),
                now + datetime.timedelta(hours = 2),
            )
            return self.cached_udk

    def generate_sas(self, guid, user_delegation_key = None):
        """Generate shared access signature

        Make a short-lived shared access signature for the cube identified by
        guid. The signature should be good for queries immediately, but is not
        suitable for storing.

        Parameters
        ----------
        guid : str
            ID for a cube
        user_delegation_key
            User delegation key, or None. defaults to self.user_delegation_key()

        Returns
        -------
        sas : str
            Shared access signature

        Examples
        --------
        >>> bauth = simple_blobstore_auth(resource)
        >>> sas = bauth.generate_sas(guid)
        >>> requests.post(f'{host}/graphql?{sas}', body)
        """
        if user_delegation_key is None:
            user_delegation_key = self.user_delegation_key()

        now = datetime.datetime.utcnow()
        fivemin = datetime.timedelta(minutes = 5)
        return azblob.generate_container_sas(
            account_name        = self.acc,
            container_name      = guid,
            user_delegation_key = user_delegation_key,
            permission          = 'r',
            start               = now - fivemin,
            expiry              = now + fivemin,
        )
