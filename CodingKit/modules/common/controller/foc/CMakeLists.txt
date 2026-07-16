cmake_minimum_required(VERSION 3.16)
project(foc_controller C)

set(CMAKE_C_STANDARD 99)
set(CMAKE_C_STANDARD_REQUIRED ON)
set(PID_CONTROLLER_DIR "${CMAKE_CURRENT_LIST_DIR}/../pid")

add_library(foc_controller
    src/foc_math.c
    src/foc_svpwm.c
    src/foc.c
    ${PID_CONTROLLER_DIR}/src/pid.c
)

target_include_directories(foc_controller PUBLIC
    src
    ${PID_CONTROLLER_DIR}/src
)
