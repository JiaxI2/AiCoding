#include "../motor_bw_lpf.h"

#include <math.h>
#include <stdio.h>

static int near_float(float a, float b, float tol)
{
    float d = fabsf(a - b);
    return d <= tol;
}

int main(void)
{
    motor_bw_lpf2_t f;
    motor_bw_lpf_status_t status;
    float y = 0.0f;
    int i;

    status = MotorBwLpf2_Init(&f, 20000.0f, 2000.0f, 0.0f);
    if(status != MOTOR_BW_LPF_OK)
    {
        printf("init failed: %d\n", (int)status);
        return 1;
    }

    /* Step response should converge to 1.0 with DC gain close to 1. */
    for(i = 0; i < 200; ++i)
    {
        y = MotorBwLpf2_Update(&f, 1.0f);
    }

    if(!near_float(y, 1.0f, 0.001f))
    {
        printf("dc gain failed: y=%f\n", y);
        return 2;
    }

    /* Invalid cutoff should disable configuration and make Update bypass input. */
    status = MotorBwLpf2_SetCutoff(&f, 10000.0f);
    if(status != MOTOR_BW_LPF_ERR_PARAM)
    {
        printf("range check failed: %d\n", (int)status);
        return 3;
    }

    y = MotorBwLpf2_Update(&f, 2.0f);
    if(!near_float(y, 2.0f, 0.00001f))
    {
        printf("bypass failed: y=%f\n", y);
        return 4;
    }

    printf("motor_bw_lpf tests passed\n");
    return 0;
}
