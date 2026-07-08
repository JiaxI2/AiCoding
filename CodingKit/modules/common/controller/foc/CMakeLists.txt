cmake_minimum_required(VERSION 3.16)
project(aicoding_foc_controller C)

set(CMAKE_C_STANDARD 99)
set(CMAKE_C_STANDARD_REQUIRED ON)
set(AICODING_PID_DIR "${CMAKE_CURRENT_LIST_DIR}/../pid")

add_library(aicoding_foc_controller
    src/foc_math.c
    src/foc_svpwm.c
    src/foc_angle.c
    src/foc_motion.c
    src/foc.c
    ${AICODING_PID_DIR}/src/pid.c
)

target_include_directories(aicoding_foc_controller PUBLIC
    src
    ${AICODING_PID_DIR}/src
)
