cmake_minimum_required(VERSION 3.21)

project(weftlink_cpp VERSION 0.4.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

find_package(Qt6 6.5 REQUIRED COMPONENTS Core Network Widgets Test)
qt_standard_project_setup()

if (MSVC)
    add_compile_options(/utf-8)
endif()

add_library(weftclient STATIC
    weftclient/frame_codec.cpp
    weftclient/weft_client.cpp
)
target_include_directories(weftclient PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})
target_link_libraries(weftclient PUBLIC Qt6::Core Qt6::Network)
target_compile_definitions(weftclient PUBLIC WEFTLINK_CLIENT_VERSION="${PROJECT_VERSION}")

qt_add_executable(weftlink-gui WIN32
    gui/main.cpp
    gui/main_window.cpp
    gui/pair_dialog.cpp
)
target_link_libraries(weftlink-gui PRIVATE weftclient Qt6::Widgets)

qt_add_executable(weftlink-tui
    tui/main.cpp
    tui/terminal.cpp
    tui/tui_app.cpp
)
target_link_libraries(weftlink-tui PRIVATE weftclient Qt6::Core)

enable_testing()
qt_add_executable(weftclient_tests
    tests/frame_codec_test.cpp
)
target_link_libraries(weftclient_tests PRIVATE weftclient Qt6::Test)
add_test(NAME weftclient_tests COMMAND weftclient_tests)
