"""
"""
import xarray as xa

from . import decoder

def splitindex(ndims, index):
    shape = index[:ndims]
    index = index[ndims:]
    for k in shape:
        yield index[:k]
        index = index[k:]

def xarray(decoded):
    """Unpack a decoded response into an xarray

    Returns
    -------
    a : xarray.DataArray
    """
    head, d = decoded
    # copy the dict so that this function can do destructive operations (on
    # the dict itself) without leaking the effects
    d = dict(d)

    ndims  = head.ndims
    labels = head.labels
    index  = [x for x in splitindex(ndims, head.index)]

    function = head.function
    data   = d.pop(head.attrs[0])
    coords = {}
    attrs  = {}
    aname  = None

    if function == decoder.functionid.slice:
        # For slices, one of the dimensions (in, cross, depth/time) should
        # be 1. There could be some awkward cases for absurdly thin cubes
        # (with dimension-of-one) which should be tested and accounted for.
        dims = []
        for ndim, name, indices in zip(data.shape, labels, index):
            if ndim > 1:
                dims.append(name)
                coords[name] = (name, indices)
            else:
                attrs[name] = indices[0]
                aname = f'{name} slice {indices[0]}'

        data = data.squeeze()
        # All other attributes describe the x/y plane
        for attr, array in d.items():
            array = array.squeeze()
            coords[attr] = (dims[:array.ndim], array.squeeze())

    elif function == decoder.functionid.curtain:
        dims = ['x, y', 'x, y', labels[-1]]
        for name, indices, dim in zip(labels, index, dims):
            coords[name] = (dim, indices)

        aname = 'curtain'
        dims.pop(0)
        for attr, array in d.items():
            coords[attr] = (dims[0], array.squeeze())

    else:
        raise RuntimeError(f'bad message; unknown function {function}')

    return xa.DataArray(
        data   = data,
        dims   = dims,
        name   = aname,
        coords = coords,
        attrs  = attrs,
    )
