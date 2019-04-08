#include <algorithm>
#include <cmath>
#include <cstdlib>
#include <fstream>
#include <iomanip>
#include <iostream>

#include <nlohmann/json.hpp>
#include <clara/clara.hpp>

using json = nlohmann::json;

namespace {

clara::ParserResult natural( int x, int& var, const std::string& name ) {
    using namespace clara;
    if (x <= 0)
        return ParserResult::runtimeError( "expected " + name + " > 0" );

    var = x;
    return ParserResult::ok( ParseResultType::Matched );
}

struct config {
    int xs = 0;
    int ys = 0;
    int zs = 0;

    int fragment_xs = 0;
    int fragment_ys = 0;
    int fragment_zs = 0;

    bool help = false;

    std::string bin;
    std::string outputdir = "";
    std::string basename = "shatter";

    clara::Parser cli() {
        using namespace clara;

        const auto checkx = [&]( int x ) { return natural( x, xs, "xs" ); };
        const auto checky = [&]( int y ) { return natural( y, ys, "ys" ); };
        const auto checkz = [&]( int z ) { return natural( z, zs, "zs" ); };

        // TODO: verify that frag-len < global-len
        const auto fragmentx = [&]( int x ) { return natural( x, fragment_xs, "xs" ); };
        const auto fragmenty = [&]( int y ) { return natural( y, fragment_ys, "ys" ); };
        const auto fragmentz = [&]( int z ) { return natural( z, fragment_zs, "zs" ); };

        return ExeName( bin )
            | Arg( checkx, "xs" )
                 ( "Samples in X direction" )
            | Arg( checky, "ys" )
                 ( "Samples in Y direction" )
            | Arg( checkz, "zs" )
                 ( "Samples in Z direction" )
            | Arg( fragmentx, "fragment xs" )
                 ( "Samples in X direction in fragment" )
            | Arg( fragmenty, "fragment ys" )
                 ( "Samples in Y direction in fragment" )
            | Arg( fragmentz, "fragment zs" )
                 ( "Samples in Z direction in fragment" )
            | Opt( outputdir, "output-dir" )
                 ["-o"]["--output-dir"]
                 ( "output directory")
            | Opt( basename, "basename" )
                 ["-b"]["--basename"]
                 ( "base name")
            | Help( this->help )
        ;
    }
};

struct dimension {
    std::size_t x;
    std::size_t y;
    std::size_t z;
};

std::vector< float > fragment( dimension local,
                               dimension local_len,
                               dimension global_len) {
    std::vector< float > out;
    out.reserve( local_len.x * local_len.y * local_len.z );

    for (std::size_t x = local.x; x < local.x + local_len.x; ++x)
    for (std::size_t y = local.y; y < local.y + local_len.y; ++y)
    for (std::size_t z = local.z; z < local.z + local_len.z; ++z)
    {
        const auto v = x * global_len.y * global_len.z
                     + y * global_len.z
                     + z
                     ;
        out.push_back( v * 0.1 );
    }

    return out;
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

    if (!cfg.xs || !cfg.ys || !cfg.zs) {
        std::cerr << cli << '\n';
        std::exit( EXIT_FAILURE );
    }

    if (!cfg.fragment_xs || !cfg.fragment_ys || !cfg.fragment_zs) {
        std::cerr << cli << '\n';
        std::exit( EXIT_FAILURE );
    }

    json manifest;

    manifest["cube-xs"] = cfg.xs;
    manifest["cube-ys"] = cfg.ys;
    manifest["cube-zs"] = cfg.zs;

    manifest["fragment-xs"] = cfg.fragment_xs;
    manifest["fragment-ys"] = cfg.fragment_ys;
    manifest["fragment-zs"] = cfg.fragment_zs;

    manifest["basename"] = cfg.basename;

    dimension num_fragments {
        std::size_t( std::ceil( double(cfg.xs) / cfg.fragment_xs ) ),
        std::size_t( std::ceil( double(cfg.ys) / cfg.fragment_ys ) ),
        std::size_t( std::ceil( double(cfg.zs) / cfg.fragment_zs ) ),
    };

    dimension global_len {
        std::size_t( cfg.xs ),
        std::size_t( cfg.ys ),
        std::size_t( cfg.zs ),
    };

    dimension local_len {
        std::size_t( cfg.fragment_xs ),
        std::size_t( cfg.fragment_ys ),
        std::size_t( cfg.fragment_zs ),
    };

    const auto outdir = cfg.outputdir.empty()
                      ? std::string()
                      : cfg.outputdir + "/"
                      ;
    for (std::size_t x = 0; x < num_fragments.x; ++x)
    for (std::size_t y = 0; y < num_fragments.y; ++y)
    for (std::size_t z = 0; z < num_fragments.z; ++z)
    {
        dimension local { x * cfg.fragment_xs,
                          y * cfg.fragment_ys,
                          z * cfg.fragment_zs };
        auto v = fragment( local, local_len, global_len );

        const std::string fname = outdir
                                + cfg.basename
                                + "-" + std::to_string( x * cfg.fragment_xs )
                                + "-" + std::to_string( y * cfg.fragment_ys )
                                + "-" + std::to_string( z * cfg.fragment_zs )
                                + ".f32"
                                ;
        std::ofstream fs{ fname, std::ios_base::binary };
        if (!fs) {
            std::cerr << "unable to open file " << fname
                      << ". non-existent directory?\n"
                      ;
            std::exit( EXIT_FAILURE );
        }
        fs.write( (const char*) v.data(), sizeof(float) * v.size() );
    }

    const auto fname = outdir + cfg.basename + ".manifest";
    std::ofstream fs{ fname };
    if (!fs) {
        std::cerr << "unable to open manifest " << fname << "\n";
        std::exit( EXIT_FAILURE );
    }
    fs << std::setw( 4 ) << manifest << "\n";
}
