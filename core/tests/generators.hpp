#include <random>

#include <catch/catch.hpp>

template < typename T >
struct IjkGenerator : Catch::Generators::IGenerator< T > {

    std::minstd_rand m_rand;
    std::uniform_int_distribution< std::size_t > x_range;
    std::uniform_int_distribution< std::size_t > y_range;
    std::uniform_int_distribution< std::size_t > z_range;

    T p;

    IjkGenerator():
        m_rand( std::random_device{}() ),
        x_range(),
        y_range(),
        z_range()
    {
        next();
    }

    IjkGenerator(std::size_t x_max, std::size_t y_max, std::size_t z_max) :
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

    T const& get() const {
        return p;
    }
};

using PointGenerator = IjkGenerator< sc::point >;
using PointGeneratorWrapper = Catch::Generators::GeneratorWrapper< sc::point >;

PointGeneratorWrapper random_points() {
    return PointGeneratorWrapper(std::make_unique< PointGenerator >());
}

PointGeneratorWrapper random_points( std::size_t x_max,
                                     std::size_t y_max,
                                     std::size_t z_max ) {
    auto gen = std::make_unique< PointGenerator >(x_max, y_max, z_max);
    return PointGeneratorWrapper(std::move(gen));
}

using DimensionGenerator = IjkGenerator< sc::dimension >;
using DimensionGeneratorWrapper = Catch::Generators::GeneratorWrapper< sc::dimension >;

DimensionGeneratorWrapper random_dimensions() {
    return DimensionGeneratorWrapper(std::make_unique< DimensionGenerator >());
}

DimensionGeneratorWrapper random_dimensions(std::size_t x_max,
                                            std::size_t y_max,
                                            std::size_t z_max) {
    auto gen = std::make_unique< DimensionGenerator >(x_max, y_max, z_max);
    return DimensionGeneratorWrapper(std::move(gen));
}
