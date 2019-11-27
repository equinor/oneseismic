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
