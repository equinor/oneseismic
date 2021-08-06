#define CATCH_CONFIG_RUNNER
#include <catch/catch.hpp>

int main(int argc, char** argv) {
    Catch::Session session;
    auto cli = session.cli();
    session.cli(cli);
    return session.run(argc, argv);
}
