#include <random>

#include <catch/catch.hpp>

struct PointGenerator : Catch::Generators::IGenerator< sc::point > {

    std::minstd_rand m_rand;
    std::uniform_int_distribution< std::size_t > x_range;
    std::uniform_int_distribution< std::size_t > y_range;
    std::uniform_int_distribution< std::size_t > z_range;

    sc::point p;

    PointGenerator():
        m_rand( std::random_device{}() ),
        x_range(),
        y_range(),
        z_range()
    {
        next();
    }

    PointGenerator( std::size_t x_max,
                    std::size_t y_max,
                    std::size_t z_max ):
        m_rand( std::random_device{}() ),
        x_range( 0, x_max ),
        y_range( 0, y_max ),
        z_range( 0, z_max )
    {
        next();
    }

    bool next() override {
        p = { x_range(m_rand), y_range(m_rand), z_range(m_rand) };
        return true;
    }

    sc::point const& get() const {
        return p;
    }
};

using PointGeneratorWrapper = Catch::Generators::GeneratorWrapper< sc::point >;

PointGeneratorWrapper random_points() {
  using Catch::Generators::IGenerator;
  return PointGeneratorWrapper( std::make_unique< PointGenerator >() );
}

PointGeneratorWrapper random_points( std::size_t x_max,
                                     std::size_t y_max,
                                     std::size_t z_max ) {
    using Catch::Generators::IGenerator;
    return PointGeneratorWrapper(std::make_unique< PointGenerator >( x_max,
                                                                     y_max,
                                                                     z_max ));
}

PointGeneratorWrapper random_dimensions() {
    return random_points();
}

PointGeneratorWrapper random_dimensions( std::size_t x_max,
                                         std::size_t y_max,
                                         std::size_t z_max ) {
    return random_points( x_max, y_max, z_max );
}
