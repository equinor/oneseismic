#include <algorithm>
#include <cerrno>
#include <cstdlib>
#include <chrono>
#include <fstream>
#include <iostream>
#include <map>
#include <string>
#include <system_error>
#include <vector>

#include <nlohmann/json.hpp>
#include <clara/clara.hpp>
#include <mio/mio.hpp>

using json = nlohmann::json;

namespace {

struct config {
    bool help = false;

    std::string bin;
    std::string manifest;
    std::string surface;
    bool timing = false;

    clara::Parser cli() {
        using namespace clara;

        return ExeName( bin )
            | Arg( manifest, "manifest" )
                 ( "Manifest" )
            | Arg( surface, "surface" )
                 ( "Surface" )
            | Opt( timing )
                 ( "Writing timing report" )
                 ["--time"]["-t"]
            | Help( this->help )
        ;
    }
};

struct point {
    int x;
    int y;
    int z;

    bool operator < ( const point& rhs ) const noexcept (true) {
        if (this->x < rhs.x) return true;
        if (this->y < rhs.y) return true;
        if (this->z < rhs.z) return true;
        return false;
    }
};

void throw_errno() {
    auto errc = static_cast< std::errc >( errno );
    throw std::system_error( std::make_error_code( errc ) );
}

std::vector< point > readsurface( const std::string& path ) {
    mio::mmap_source file( path );

    if (file.size() % (sizeof(int) * 3) != 0)
        throw std::runtime_error( "truncated surface" );

    const int elems = file.size() / (sizeof(int) * 3);
    std::vector< point > xs;
    xs.reserve( elems );

    const char* itr = static_cast< const char* >( file.data() );
    for (int i = 0; i < elems; ++i) {
        point p;
        std::memcpy( &p.x, itr + 0, 4 );
        std::memcpy( &p.y, itr + 4, 4 );
        std::memcpy( &p.z, itr + 8, 4 );
        itr += 12;
        xs.push_back( p );
    }

    return xs;
}

std::map< point, std::vector< int > > bin( point fragment_size,
                                           point cube_size,
                                           const std::vector< point >& xs ) {
    std::map< point, std::vector< int > > ret;
    for (const auto& p : xs) {
        point root {
            (p.x / fragment_size.x) * fragment_size.x,
            (p.y / fragment_size.y) * fragment_size.y,
            (p.z / fragment_size.z) * fragment_size.z,
        };

        point local = {
            p.x % fragment_size.x,
            p.y % fragment_size.y,
            p.z % fragment_size.z,
        };

        int pos = local.x * (fragment_size.y * fragment_size.z)
                + local.y *  fragment_size.z
                + local.z
                ;
        ret[root].push_back(pos);
    }

    for (auto& kv : ret)
        std::sort( kv.second.begin(), kv.second.end() );

    return ret;
}

}

int main( int args, char** argv ) {
    config cfg;
    auto cli = cfg.cli();

    auto result = cli.parse( clara::Args( args, argv ) );

    if (cfg.help) {
        std::cerr << cli << "\n";
        std::exit( EXIT_SUCCESS );
    }

    if (!result) {
        std::cerr << result.errorMessage() << '\n';
        std::cerr << "usage: " << cli << '\n';
        std::exit( EXIT_FAILURE );
    }

    json manifest;
    std::ifstream( cfg.manifest ) >> manifest;

    point fragment_size {
        manifest["fragment-xs"].get< int >(),
        manifest["fragment-ys"].get< int >(),
        manifest["fragment-zs"].get< int >(),
    };

    point cube_size {
        manifest["cube-xs"].get< int >(),
        manifest["cube-ys"].get< int >(),
        manifest["cube-zs"].get< int >(),
    };

    auto start_time = std::chrono::system_clock::now();

    const auto surface = readsurface( cfg.surface );
    auto surface_time = std::chrono::system_clock::now();

    const auto bins = bin( fragment_size, cube_size, surface );
    auto bin_time = std::chrono::system_clock::now();

    decltype (bins.begin()) itr;
    #pragma omp parallel for
    #pragma omp single nowait
    {
    for (itr = bins.begin(); itr != bins.end(); ++itr) {
        #pragma omp task firstprivate(itr)
        {
        const auto& key = itr->first;
        const auto& val = itr->second;
        const std::string path = manifest["basename"].get< std::string >()
                               + "-" + std::to_string( key.x )
                               + "-" + std::to_string( key.y )
                               + "-" + std::to_string( key.z )
                               + ".f32"
                               ;
        mio::mmap_source file( path );

        const char* ptr = static_cast< const char* >( file.data() );

        std::vector< float > xs;
        xs.reserve( val.size() );
        for (const auto& off : val) {
            float f;
            std::memcpy( &f, ptr + off * 4, 4 );
            xs.push_back( f );
        }

        std::string outpath = "/data/js/out-" + path;
        std::ofstream ofs( outpath, std::ios::binary );
        ofs.write( (char*)xs.data(), xs.size() * sizeof(float));
        }
    }

    } // omp

    auto end_time = std::chrono::system_clock::now();

    if (cfg.timing) {
        using namespace std::chrono;
        auto surf =  duration_cast< milliseconds >(surface_time - start_time);
        auto bin =   duration_cast< milliseconds >(bin_time - surface_time);
        auto read =  duration_cast< milliseconds >(end_time - bin_time);
        auto total = duration_cast< milliseconds >(end_time - start_time);

        std::cout << "Timing report: \n"
                  << "Parsing surface: "    << surf.count()  << "ms\n"
                  << "Binning surface: "    << bin.count()   << "ms\n"
                  << "Reading surface: "    << read.count()  << "ms\n"
                  << "Total elapsed time: " << total.count() << "ms\n"
        ;
    }
}
