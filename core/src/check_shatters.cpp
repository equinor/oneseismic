#include <algorithm>
#include <cerrno>
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <map>
#include <string>
#include <system_error>
#include <vector>

#include <sys/mman.h>
#include <sys/types.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/stat.h>

#include <nlohmann/json.hpp>
#include <clara/clara.hpp>
#include <segyio/segyio.hpp>

using json = nlohmann::json;

namespace {

struct config {
    bool help = false;

    std::string bin;
    std::string manifest;
    std::string segyfile;
    std::string source_dir = "./";
    int xlbyte = 193;
    int ilbyte = 187;

    clara::Parser cli() {
        using namespace clara;

        return ExeName( bin )
            | Arg( manifest, "manifest" )
                 ( "Manifest" )
            | Arg( segyfile, "segyfile" )
                 ( "Segy file" )
            | Opt( source_dir, "source directory" )
                 ["-s"]["--source-dir"]
            | Opt( xlbyte, "xlbyte" )
                 ["-x"]["--xlbyte"]
            | Opt( ilbyte, "ilbyte" )
                 ["-i"]["--ilbyte"]
            | Help( this->help )
        ;
    }
};

using segy = segyio::basic_volume<>;

struct point {
    int x;
    int y;
    int z;

    bool operator!=( const point& rhs ) {
        return not( x == rhs.x and y == rhs.y and z == rhs.z );
    }

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


point fragment_id( point global_coords, point fragment_size ) {
    const int ux = (global_coords.x / fragment_size.x) * fragment_size.x;
    const int uy = (global_coords.y / fragment_size.y) * fragment_size.y;
    const int uz = (global_coords.z / fragment_size.z) * fragment_size.z;

    return { ux, uy, uz };
}

int fragment_offset( point global_coords, point fragment_size ) {
    const int lx = global_coords.x % fragment_size.x;
    const int ly = global_coords.y % fragment_size.y;
    const int lz = global_coords.z % fragment_size.z;

    return lz + fragment_size.z*ly + fragment_size.y*fragment_size.z*lx;
}

size_t getFilesize(const char* filename) {
    struct stat st;
    stat(filename, &st);
    return st.st_size;
}

void load_from_fragment( const std::vector< point >& glbl_coordinates,
                         point frgmnt_id,
                         point fragment_size,
                         const std::string& base_filename,
                         std::vector< float >& out ) {

    const std::string fname = base_filename
                            + "-" + std::to_string( frgmnt_id.x )
                            + "-" + std::to_string( frgmnt_id.y )
                            + "-" + std::to_string( frgmnt_id.z )
                            + ".f32";

    size_t filesize = getFilesize(fname.c_str());
    int fd = open(fname.c_str(), O_RDONLY, 0);
    void* mmappedData = mmap(NULL, filesize,
                             PROT_READ, MAP_PRIVATE | MAP_POPULATE, fd, 0);

    for( int i = 0; i < glbl_coordinates.size(); ++i ) {
        point glbl = glbl_coordinates[i];

        if( fragment_id( glbl, fragment_size ) != frgmnt_id )
            continue;

        const int offset = fragment_offset( glbl,
                                            fragment_size );
        float t = *((float*)mmappedData + offset);
        out[i] = t;
    }

    munmap(mmappedData, filesize);
    close(fd);
}

std::vector< float > retrieve_data( const std::vector< point >& xs,
                                    point fragment_size,
                                    const std::string& base_filename ) {

    std::vector< float > data( xs.size() );

    std::set< point > subcubes;

    for( point p : xs )
        subcubes.insert( fragment_id( p, fragment_size ) );

    #pragma omp parallel
    {
        #pragma omp single
        {
            for( point subcube : subcubes ) {
                #pragma omp task
                load_from_fragment( xs, subcube, fragment_size,
                                    base_filename, data );
            }
        }
    }

    return data;
}

std::vector< float > get_trace( int traceno,
                                point fragment_size,
                                point cubesize,
                                const std::string& base_filename ) {
    std::vector< point > xs( cubesize.z );

    const int x = traceno / cubesize.y;
    const int y = traceno % cubesize.y;

    int z = 0;
    std::generate_n(xs.begin(), cubesize.z, [&](){ return point{x, y, z++}; });

    return retrieve_data( xs, fragment_size, base_filename );
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

    const std::string source_dir = cfg.source_dir + "/";

    json manifest;
    std::ifstream( source_dir + cfg.manifest ) >> manifest;

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

    std::string base_filename = source_dir
                              + manifest["basename"].get< std::string >();

    segy cube {
      segyio::path{ cfg.segyfile },
      segyio::config{}.with(segyio::ilbyte{ cfg.ilbyte })
                      .with(segyio::xlbyte{ cfg.xlbyte }),
    };

    std::vector< float > trace( cube.samplecount() );

    for (int trc = 0; trc < cube.tracecount(); trc++ ) {
        cube.get( trc, trace.begin() );
        auto shatter_trace = get_trace( trc, fragment_size,
                                        cube_size, base_filename );
        if( trace != shatter_trace ) {
            std::cout << "Mismatch at trace: " << trc << "\n";
        }
    }
}
