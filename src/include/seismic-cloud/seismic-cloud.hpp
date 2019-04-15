#ifndef SEISMIC_CLOUD_HPP
#define SEISMIC_CLOUD_HPP

namespace sc {

struct point {
    std::size_t x;
    std::size_t y;
    std::size_t z;

    bool operator < (const point& rhs) const noexcept (true) {
        if (this->x < rhs.x) return true;
        if (this->y < rhs.y) return true;
        if (this->z < rhs.z) return true;
        return false;
    }
};

}

#endif //SEISMIC_CLOUD_HPP
