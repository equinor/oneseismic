#include <vector>
#include <iostream>
#include <fstream>

#include <clara/clara.hpp>
#include <nlohmann/json.hpp>

#include <seismic-cloud/seismic-cloud.hpp>

using json = nlohmann::json;

struct config {
    bool help = false;

    std::string bin;
    std::string manifest;
    std::string surface;
    std::string input_dir = "./";

    clara::Parser cli() {
        using namespace clara;

        return ExeName( bin )
            | Arg( manifest, "manifest" )
                 ( "Manifest" )
            | Arg( surface, "surface" )
                 ( "Surface" )
            | Opt( input_dir, "Input directory" )
                 ["--input-dir"]["-i"]
            | Help( this->help )
        ;
    }
};

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
    std::ifstream( cfg.input_dir + "/" + cfg.manifest ) >> manifest;

    sc::point cube_size {
        manifest["cube-xs"].get< std::size_t >(),
        manifest["cube-ys"].get< std::size_t >(),
        manifest["cube-zs"].get< std::size_t >(),
    };

    std::vector< std::uint64_t > offsets;
    std::vector< float > values;

    while (true) {
        std::uint64_t offset;
        float value;

        std::cin.read( (char*)&offset, sizeof(std::uint64_t) );
        std::cin.read( (char*)&value, sizeof(float) );

        if (std::cin.eof()) break;

        offsets.push_back(offset);
        values.push_back(value);
    }

    std::uint64_t result_size = offsets.size();

    std::vector< int > xs( result_size );
    std::vector< int > ys( result_size );
    std::vector< int > zs( result_size );

    for (int i = 0; i < result_size; ++i) {
        std::uint64_t offset = offsets[i];
        float v = values[i];

        float expected = 0.1 * offset;

        if ( std::abs(expected - v) > 1e-5 ) std::exit( EXIT_FAILURE );

        int x = offset / (cube_size.y * cube_size.z);
        int y = (offset % (cube_size.y * cube_size.z)) / cube_size.z;
        int z = (offset % (cube_size.y * cube_size.z)) % cube_size.z;

        xs[i] = x;
        ys[i] = y;
        zs[i] = z;
    }

    std::ifstream srfc( cfg.surface );

    srfc.seekg( 0, srfc.end );
    auto size = srfc.tellg();
    srfc.seekg( 0, srfc.beg );

    std::vector< char > points( size );
    srfc.read( points.data(), size );

    std::size_t expected_size = points.size() / ( sizeof(std::int32_t) * 3 );

    std::vector< sc::point > surface( expected_size );

    [&surface] (char* ptr) {
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

    std::vector< int > expected_xs( expected_size );
    std::vector< int > expected_ys( expected_size );
    std::vector< int > expected_zs( expected_size );

    for (int i = 0; i < expected_size; ++i) {
        expected_xs[i] = surface[i].x;
        expected_ys[i] = surface[i].y;
        expected_zs[i] = surface[i].z;
    }

    std::sort( xs.begin(), xs.end() );
    std::sort( ys.begin(), ys.end() );
    std::sort( zs.begin(), zs.end() );

    std::sort( expected_xs.begin(), expected_xs.end() );
    std::sort( expected_ys.begin(), expected_ys.end() );
    std::sort( expected_zs.begin(), expected_zs.end() );

    if( expected_xs.size() != xs.size() ) std::exit( EXIT_FAILURE );

    for (int i = 0; i < expected_xs.size(); ++i ) {
        if ( xs[i] != expected_xs[i] ) {
          std::cout << i << "\n";
          std::cout << xs[i] << "\n";
          std::cout << expected_xs[i] << "\n\n";
          std::exit( EXIT_FAILURE );
        }
        if ( ys[i] != expected_ys[i] ) {
          std::cout << i << "\n";
          std::cout << ys[i] << "\n";
          std::cout << expected_ys[i] << "\n\n";
          std::exit( EXIT_FAILURE );
        }
        if ( zs[i] != expected_zs[i] ) {
          std::cout << i << "\n";
          std::cout << zs[i] << "\n";
          std::cout << expected_zs[i] << "\n\n";
          std::exit( EXIT_FAILURE );
        }
    }
}
