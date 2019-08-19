#include <fstream>
#include <mio/mio.hpp>

#include <pybind11/pybind11.h>
#include <pybind11/stl.h>

#include <seismic-cloud/seismic-cloud.hpp>

namespace py = pybind11;

std::vector< sc::point > read_surface( const std::string& filename ) {

    std::ifstream f( filename );

    f.seekg (0, f.end);
    std::size_t fsize = f.tellg();
    f.seekg (0, f.beg);

    std::vector< char > points( fsize );
    f.read( points.data(), points.size() );

    std::size_t size = points.size() / ( sizeof(std::int32_t) * 3 );

    std::vector< sc::point > surface( size );

    [&surface] (const char* ptr) {
        for (auto& p : surface) {
            std::int32_t x, y, z;
            std::memcpy(&x, ptr, sizeof(x));
            ptr += sizeof(x);
            std::memcpy(&y, ptr, sizeof(y));
            ptr += sizeof(y);
            std::memcpy(&z, ptr, sizeof(z));
            ptr += sizeof(z);

            p.x = x;
            p.y = y;
            p.z = z;
        }
    }(points.data());

    return surface;
}

void surface( const std::string& base_dir,
              const std::string& surface_name,
              const std::string& basename,
              int cubexs,
              int cubeys,
              int cubezs,
              int fragmentxs,
              int fragmentys,
              int fragmentzs,
              py::object& reply ) {
    sc::dimension cube_size;
    cube_size.x = cubexs;
    cube_size.y = cubeys;
    cube_size.z = cubezs;

    sc::dimension fragment_size;
    fragment_size.x = fragmentxs;
    fragment_size.y = fragmentys;
    fragment_size.z = fragmentzs;

    const auto surface = read_surface( base_dir + "/" + surface_name );

    // TODO: error detection (and recovery?) when loading surface
    //if( <surface reading fails> ) return Status::FAILED_PRECONDITION;

    const auto bins = sc::bin(fragment_size, cube_size, surface);

    #pragma omp parallel for
    for (std::size_t i = 0; i < bins.keys.size(); ++i) {
        const auto bin = bins.at(i);
        const auto& key = bin.key;
        const std::string path = basename
                               + "-" + std::to_string( key.x )
                               + "-" + std::to_string( key.y )
                               + "-" + std::to_string( key.z )
                               + ".f32"
                               ;
        mio::mmap_source file( base_dir + "/" + path );

        const auto bin_size = std::distance(bin.begin(), bin.end());

        std::vector< std::int64_t > output_i;
        std::vector< float > output_v;
        output_i.reserve(bin_size);
        output_v.reserve(bin_size);

        const char* in = static_cast< const char* >(file.data());

        for (const auto off : bin) {
            const std::uint64_t global_offset =
                sc::local_to_global(off, fragment_size, cube_size, key);

            float value;
            std::memcpy(&value, in + off * 4, sizeof(float));

            output_i.push_back( global_offset );
            output_v.push_back( value );
        }

        #pragma omp critical
        {
        reply.attr("i").attr("extend")( output_i );
        reply.attr("v").attr("extend")( output_v );
        }
    } // omp

    return;
}

PYBIND11_MODULE(stitch, m) {
    m.def("surface", &surface, "");
}
