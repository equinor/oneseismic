#include <algorithm>
#include <cmath>
#include <fstream>
#include <iomanip>
#include <iostream>
#include <utility>
#include <vector>

#include <nlohmann/json.hpp>
#include <clara/clara.hpp>
#include <segyio/segyio.hpp>

#include <seismic-cloud/seismic-cloud.hpp>

using json = nlohmann::json;

namespace {

struct config {
    bool help = false;

    std::string bin;
    std::string fname;
    std::string prefix = "shatter";
    std::string outputdir = "";
    int xs = 0;
    int ys = 0;
    int zs = 0;
    int xlbyte = 193;
    int ilbyte = 189;

    clara::Parser cli() {
        using namespace clara;

        return ExeName( bin )
            | Arg( fname , "volume" )
                 ( "SEG-Y volume" )
            | Arg( xs, "x count" )
            | Arg( ys, "y count" )
            | Arg( zs, "z count" )
            | Opt( prefix, "prefix" )
                 ["-p"]["--prefix"]
                 ( "Prefix" )
            | Opt( outputdir, "output-dir" )
                 ["-o"]["--output-dir"]
                 ( "output directory")
            | Opt( xlbyte, "xlbyte" )
                 ["-x"]["--xlbyte"]
            | Opt( ilbyte, "ilbyte" )
                 ["-i"]["--ilbyte"]
            | Help( this->help )
        ;
    }
};

}

using segy = segyio::basic_volume<>;

std::vector< std::pair< int, int > > cartesian( sc::dimension dim,
                                                sc::dimension cubesize ) {
    std::vector< std::pair< int, int > > cart;

    for (int x = 0; x < dim.x; ++x) {
        for (int y = 0; y < dim.y; ++y) {
            cart.emplace_back( x * cubesize.x, cubesize.y * y );
        }
    }

    return cart;
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

    if (!cfg.xs || !cfg.ys || !cfg.zs) {
        std::cerr << cli << '\n';
        std::exit( EXIT_FAILURE );
    }

    const auto outdir = cfg.outputdir.empty()
                      ? std::string()
                      : cfg.outputdir + "/"
                      ;

    segy cube {
      segyio::path{ cfg.fname },
      segyio::config{}.with(segyio::ilbyte{ cfg.ilbyte })
                      .with(segyio::xlbyte{ cfg.xlbyte }),
    };

    if (!(cube.sorting() == segyio::sorting::iline())) {
        const std::string msg = "shatter only support inline sorted files";
        throw std::invalid_argument( msg );
    }

    const auto inlinecount    = cube.inlinecount();
    const auto crosslinecount = cube.crosslinecount();
    const auto samplecount    = cube.samplecount();

    sc::dimension num_fragments {
        int( std::ceil( double(inlinecount)    / cfg.xs ) ),
        int( std::ceil( double(crosslinecount) / cfg.ys ) ),
        int( std::ceil( double(samplecount)    / cfg.zs ) ),
    };

    auto corners = cartesian( num_fragments, { cfg.xs, cfg.ys } );

    for (const auto& corner : corners) {
        const auto max_x = corner.first  + cfg.xs;
        const auto max_y = corner.second + cfg.ys;

        const auto edge_x = std::min(max_x, inlinecount);
        const auto edge_y = std::min(max_y, crosslinecount);
        std::vector< float > column;

        const auto z_len = cfg.zs * num_fragments.z;
        const auto z_pad = z_len - samplecount;

        for (auto x = corner.first;  x < max_x; ++x)
        for (auto y = corner.second; y < max_y; ++y)
        {
            if (x < edge_x && y < edge_y) {
                auto traceno = x * crosslinecount + y;
                cube.get(traceno, std::back_inserter(column));
                std::fill_n( std::back_inserter(column), z_pad, 0 );
            }
            else {
                /* pad */
                std::fill_n( std::back_inserter(column), z_len, 0 );
            }
        }

        const auto n = cfg.zs;
        for (int z = 0; z < num_fragments.z * n; z += n) {
            std::vector< float > fragment( cfg.zs * cfg.xs * cfg.ys );
            auto out = fragment.begin();

            auto zitr = column.begin() + z;
            for (int i = 0; i < cfg.xs * cfg.ys; ++i) {
                out = std::copy_n( zitr, n, out );
                zitr += z_len;
            }

            std::string fname = outdir
                              + cfg.prefix
                              + "-" + std::to_string(corner.first)
                              + "-" + std::to_string(corner.second)
                              + "-" + std::to_string(z)
                              + ".f32"
                              ;
            std::ofstream fs( fname, std::ios::binary );
            fs.write( (char*)fragment.data(), fragment.size() * sizeof(float) );
        }
    }

    json manifest;

    manifest["fragment-xs"] = cfg.xs;
    manifest["fragment-ys"] = cfg.ys;
    manifest["fragment-zs"] = cfg.zs;

    manifest["cube-xs"] = inlinecount;
    manifest["cube-ys"] = crosslinecount;
    manifest["cube-zs"] = samplecount;

    manifest["basename"] = cfg.prefix;

    const auto fname = outdir + cfg.prefix + ".manifest";
    std::ofstream fs{ fname };
    if (!fs) {
        std::cerr << "unable to open manifest " << fname << "\n";
        std::exit( EXIT_FAILURE );
    }
    fs << std::setw( 4 ) << manifest << "\n";
}
