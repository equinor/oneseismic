#include <fstream>
#include <stdlib.h>
#include <cmath>
#include <iostream>

#include <clara/clara.hpp>

void create_surface( int xs,
                     int ys,
                     float amplitude,
                     float steepness,
                     float height,
                     std::string filename) {

    std::ofstream f(filename);

    int size = xs * ys;

    f << "{\"size\": " << size << "}";

    for (int x = 0; x < xs; ++x)
    for (int y = 0; y < ys; ++y) {
        int z = amplitude * std::sin(steepness*(x+y)) + height;
        f.write( (char*)&x, sizeof(int) );
        f.write( (char*)&y, sizeof(int) );
        f.write( (char*)&z, sizeof(int) );
    }

}

struct config {
    bool help = false;

    std::string bin;
    std::string output_dir;

    clara::Parser cli() {
        using namespace clara;

        return ExeName( bin )
            | Arg( output_dir, "output dir" )
                 ( "Output directory" )
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

    create_surface(400, 300, 200, 0.005, 200,  cfg.output_dir + "surface1.i32");
    create_surface(400, 300, 200, 0.005, 600,  cfg.output_dir + "surface2.i32");
    create_surface(400, 300, 200, 0.005, 1050, cfg.output_dir + "surface3.i32");

    create_surface(400, 300, 400, 0.004, 400, cfg.output_dir + "surface4.i32");
    create_surface(400, 300, 400, 0.004, 600, cfg.output_dir + "surface5.i32");
    create_surface(400, 300, 400, 0.004, 850, cfg.output_dir + "surface6.i32");

    create_surface(400, 300, 200, 0.01, 200,  cfg.output_dir + "surface7.i32");
    create_surface(400, 300, 200, 0.01, 600,  cfg.output_dir + "surface8.i32");
    create_surface(400, 300, 200, 0.01, 1050, cfg.output_dir + "surface9.i32");
}
