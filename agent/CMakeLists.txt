cmake_minimum_required(VERSION 3.16)

project(SentriXAgent VERSION 1.0.0 LANGUAGES C)

set(CMAKE_C_STANDARD 11)

add_definitions(-D_GNU_SOURCE)

if(CMAKE_C_COMPILER_ID MATCHES "GNU|Clang")
    add_compile_options(
        -Wall
        -Wextra
        -fstack-protector-strong
        -D_FORTIFY_SOURCE=2
        -O2
    )
endif()

option(ENABLE_SANITIZERS "Enable Address and UndefinedBehavior sanitizers" OFF)
if(ENABLE_SANITIZERS)
    add_compile_options(-fsanitize=address,undefined -fno-omit-frame-pointer)
    add_link_options(-fsanitize=address,undefined)
endif()

if(NOT DEFINED SENTRIX_VERSION_STRING)
    set(SENTRIX_VERSION_STRING "v${PROJECT_VERSION}")
endif()

set(SOURCES
    src/main.c
    src/core/transport.c
    src/core/buffer.c
    src/core/checks.c
    src/core/cJSON.c
    src/platform/linux/cpu.c
    src/platform/linux/memory.c
    src/platform/linux/system.c
)

add_executable(sentrix-agent ${SOURCES})

target_include_directories(sentrix-agent PRIVATE
    include
    src/core
)

target_compile_definitions(sentrix-agent PRIVATE
    SENTRIX_VERSION_STRING="${SENTRIX_VERSION_STRING}"
)

option(SENTRIX_INSECURE_DEV_NO_TLS "Allow building sentrix-agent without OpenSSL (development only)" OFF)
option(SENTRIX_ALLOW_INSECURE_TLS "Allow SENTRIX_INSECURE_SKIP_VERIFY env var (development only)" OFF)
if(SENTRIX_ALLOW_INSECURE_TLS)
    target_compile_definitions(sentrix-agent PRIVATE SENTRIX_ALLOW_INSECURE_TLS=1)
endif()

if(SENTRIX_INSECURE_DEV_NO_TLS)
    find_package(OpenSSL QUIET)
    if(NOT OPENSSL_FOUND)
        message(WARNING "Building sentrix-agent without OpenSSL! Insecure dev build only - TLS transport disabled.")
    endif()
else()
    find_package(OpenSSL REQUIRED)
endif()

target_link_libraries(sentrix-agent m)

if(OPENSSL_FOUND)
    message(STATUS "OpenSSL found: enabling secure TLS transport for sentrix-agent")
    target_compile_definitions(sentrix-agent PRIVATE SENTRIX_ENABLE_TLS)
    target_include_directories(sentrix-agent PRIVATE ${OPENSSL_INCLUDE_DIR})
    target_link_libraries(sentrix-agent OpenSSL::SSL OpenSSL::Crypto)
else()
    message(STATUS "OpenSSL not found: agent building without native TLS support")
endif()

install(TARGETS sentrix-agent RUNTIME DESTINATION bin)

set(CPACK_GENERATOR "TGZ")
set(CPACK_PACKAGE_NAME "sentrix-agent")
set(CPACK_PACKAGE_VENDOR "SentriX")
set(CPACK_PACKAGE_VERSION "${PROJECT_VERSION}")
set(CPACK_PACKAGE_FILE_NAME "sentrix-agent-${PROJECT_VERSION}-linux-${CMAKE_SYSTEM_PROCESSOR}")

include(CPack)
